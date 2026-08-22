# Backend Security Hardening & IP Rate Limiting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hardening the Go backend against DoS, memory leaks, brute-force, and web vulnerabilities by adding an IP-level Rate Limiter middleware, WebSocket ping/pong heartbeat timeouts, strict security headers (CSP, HSTS, Permissions-Policy), payload size caps, custom Room ID sanitization, and Prometheus rate-limiting metrics.

**Architecture:** Create an `internal/middleware` package containing `ratelimit.go` and `security.go`. Update `room_manager.go` with regex validation. Add ping/pong tickers to `handler/ws.go`. Apply middleware in `cmd/server/main.go`.

**Tech Stack:** Go standard library `sync`, `net/http`, `regexp`, `golang.org/x/time/rate`, `github.com/gorilla/websocket`, `github.com/prometheus/client_golang`.

## Global Constraints

- Must pass all tests with `go test -race ./...`.
- Must not break existing WebSocket signaling flow.
- Rate limiter must automatically purge idle IP entries to prevent memory leaks.

---

### Task 1: IP Rate Limiter Package & Metrics (`internal/middleware/ratelimit.go` & `ratelimit_test.go`)

**Files:**
- Create: `backend/internal/middleware/ratelimit.go`
- Create: `backend/internal/middleware/ratelimit_test.go`
- Modify: `backend/internal/telemetry/metrics.go`

**Interfaces:**
- Produces:
  - `middleware.IPRateLimiter` (`Allow(ip string) bool`, `LimitMiddleware(r rate.Limit, b int)`)
  - Prometheus metric `ezyshare_rate_limit_exceeded_total` CounterVec

- [ ] **Step 1: Add Rate Limit Metric in `telemetry/metrics.go`**

Update `backend/internal/telemetry/metrics.go`:

```go
type Metrics struct {
	// Gauge metrics
	ActivePeers prometheus.Gauge
	ActiveRooms prometheus.Gauge
	// Counter metrics
	MessagesRelayed      *prometheus.CounterVec
	WebSocketConnections *prometheus.CounterVec
	HTTPRequestsTotal    *prometheus.CounterVec
	RoomsCreatedTotal    *prometheus.CounterVec
	RateLimitExceeded    *prometheus.CounterVec
	// Histogram metric
	HTTPRequestDuration *prometheus.HistogramVec
}
```

And in `NewMetrics`:

```go
		RateLimitExceeded: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_rate_limit_exceeded_total",
				Help: "Total number of rate limit exceeded events.",
			},
			[]string{"endpoint"},
		),
```

- [ ] **Step 2: Create `ratelimit.go`**

Create `backend/internal/middleware/ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu      sync.RWMutex
	ips     map[string]*ipLimiter
	rate    rate.Limit
	burst   int
	metrics *telemetry.Metrics
	quit    chan struct{}
}

func NewIPRateLimiter(r rate.Limit, b int, metrics *telemetry.Metrics) *IPRateLimiter {
	lim := &IPRateLimiter{
		ips:     make(map[string]*ipLimiter),
		rate:    r,
		burst:   b,
		metrics: metrics,
		quit:    make(chan struct{}),
	}

	go lim.cleanupLoop()
	return lim
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	lim, exists := i.ips[ip]
	if !exists {
		l := rate.NewLimiter(i.rate, i.burst)
		i.ips[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}

	lim.lastSeen = time.Now()
	return lim.limiter
}

func (i *IPRateLimiter) Allow(ip string) bool {
	return i.GetLimiter(ip).Allow()
}

func (i *IPRateLimiter) Stop() {
	close(i.quit)
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			i.mu.Lock()
			for ip, entry := range i.ips {
				if time.Since(entry.lastSeen) > 10*time.Minute {
					delete(i.ips, ip)
				}
			}
			i.mu.Unlock()
		case <-i.quit:
			return
		}
	}
}

func RateLimitMiddleware(limiter *IPRateLimiter, endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := telemetry.GetClientIP(r)

			if !limiter.Allow(ip) {
				if limiter.metrics != nil {
					limiter.metrics.RateLimitExceeded.WithLabelValues(endpoint).Inc()
				}
				w.Header().Set("Retry-After", "60")
				http.Error(w, "429 Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 3: Create `ratelimit_test.go`**

Create `backend/internal/middleware/ratelimit_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/middleware"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestIPRateLimiter(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	limiter := middleware.NewIPRateLimiter(rate.Every(100*time.Millisecond), 2, metrics)
	defer limiter.Stop()

	handler := middleware.RateLimitMiddleware(limiter, "test_endpoint")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1: Allow
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	// Request 2: Allow (burst 2)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	// Request 3: Exceeded Rate Limit
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec3.Code)
	}

	if val := testutil.ToFloat64(metrics.RateLimitExceeded.WithLabelValues("test_endpoint")); val != 1 {
		t.Fatalf("expected rate limit metric to be 1, got %f", val)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/middleware/... -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/middleware/ backend/internal/telemetry/
git commit -m "feat(middleware): add thread-safe IP rate limiter with prometheus metrics"
```

---

### Task 2: Strict Security Headers & MaxBytes Payload Protection (`internal/middleware/security.go`)

**Files:**
- Create: `backend/internal/middleware/security.go`
- Create: `backend/internal/middleware/security_test.go`

**Interfaces:**
- Produces:
  - `middleware.SecurityHeaders(appEnv string) func(http.Handler) http.Handler`
  - `middleware.MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler`

- [ ] **Step 1: Create `security.go`**

Create `backend/internal/middleware/security.go`:

```go
package middleware

import (
	"net/http"
)

func SecurityHeaders(appEnv string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

			// Content Security Policy
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self'; "+
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"img-src 'self' data: blob:; "+
					"connect-src 'self' wss: ws:; "+
					"object-src 'none'; "+
					"frame-ancestors 'none';")

			if appEnv == "prod" {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 2: Create `security_test.go`**

Create `backend/internal/middleware/security_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders("prod")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing X-Frame-Options")
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("missing HSTS header in prod")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("missing CSP header")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/middleware/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/middleware/
git commit -m "feat(middleware): add strict security headers (CSP, HSTS, Permissions-Policy) and MaxBytes payload guard"
```

---

### Task 3: Custom Room ID Regex Validation & Sanitization (`internal/signaling/room_manager.go`)

**Files:**
- Modify: `backend/internal/signaling/room_manager.go`
- Modify: `backend/internal/signaling/room_manager_test.go`

**Interfaces:**
- Produces:
  - `ErrInvalidRoomID` error
  - Regex validation `^[a-zA-Z0-9_-]{4,64}$` for custom room IDs

- [ ] **Step 1: Update `room_manager.go`**

Update `backend/internal/signaling/room_manager.go`:

```go
var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomIDTaken   = errors.New("custom room ID already in use")
	ErrRoomFull      = errors.New("room reached maximum peer capacity")
	ErrInvalidRoomID = errors.New("invalid custom room ID (must be 4-64 alphanumeric characters, hyphens, or underscores)")
)

var roomIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,64}$`)

func (rm *RoomManager) CreateRoom(customID, hostID string) (*domain.Room, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	roomID := customID
	isCustom := true
	if roomID == "" {
		roomID = rm.generateUUID()
		isCustom = false
	} else {
		if !roomIDRegex.MatchString(roomID) {
			return nil, ErrInvalidRoomID
		}
	}

	if _, exists := rm.rooms[roomID]; exists {
		return nil, ErrRoomIDTaken
	}

	room := domain.NewRoom(roomID, hostID)
	rm.rooms[roomID] = room

	if rm.metrics != nil {
		rm.metrics.ActiveRooms.Inc()
		roomType := "custom"
		if !isCustom {
			roomType = "uuid"
		}
		rm.metrics.RoomsCreatedTotal.WithLabelValues(roomType).Inc()
	}

	slog.Info("Room created", "room_id", roomID, "host_id", hostID, "is_custom", isCustom)
	return room, nil
}
```

- [ ] **Step 2: Update `room_manager_test.go`**

Add invalid room ID test in `room_manager_test.go`:

```go
	// Invalid custom room ID (too short / special chars) fails
	_, errInvalid := rm.CreateRoom("ab", "peer-1")
	if errInvalid != signaling.ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID, got %v", errInvalid)
	}

	_, errInjection := rm.CreateRoom("<script>alert(1)</script>", "peer-1")
	if errInjection != signaling.ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID for XSS payload, got %v", errInjection)
	}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/signaling/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/signaling/
git commit -m "feat(signaling): add custom Room ID regex validation and XSS/sanitization guard"
```

---

### Task 4: WebSocket Heartbeat Ping/Pong & Dead Connection Pruning (`internal/handler/ws.go`)

**Files:**
- Modify: `backend/internal/handler/ws.go`

**Interfaces:**
- Produces:
  - 30-second Ping ticker to prune dead TCP sockets and prevent zombie connection memory leaks.

- [ ] **Step 1: Update `ws.go` with Ping/Pong Ticker**

Update connection loop in `backend/internal/handler/ws.go`:

```go
	pingPeriod := 30 * time.Second
	pongWait := 60 * time.Second
	writeWait := 10 * time.Second

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()
```

- [ ] **Step 2: Run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v -race`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(handler): add WebSocket ping/pong heartbeat ticker to prune zombie TCP connections"
```

---

### Task 5: Server Integration & Comprehensive Verification (`cmd/server/main.go`)

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/cmd/server/server_test.go`

- [ ] **Step 1: Wire Rate Limiter & Security Middlewares in `main.go`**

Update `backend/cmd/server/main.go`:

```go
	// IP Rate Limiters
	apiLimiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/60), 20, metrics) // 60 req/min
	wsLimiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/10), 5, metrics)   // 10 WS upgrades/min
	defer apiLimiter.Stop()
	defer wsLimiter.Stop()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SecurityHeaders(cfg.AppEnv))
	r.Use(middleware.MaxBytesMiddleware(4 * 1024)) // Cap HTTP bodies at 4KB

	// CORS...
	// HTTP Logging...

	r.With(middleware.RateLimitMiddleware(wsLimiter, "ws_upgrade")).Get("/ws", wsHandler.ServeWS)
	r.With(middleware.RateLimitMiddleware(apiLimiter, "healthz")).Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.With(middleware.RateLimitMiddleware(apiLimiter, "ice_servers")).Get("/api/v1/ice-servers", handler.HandleICEServers)
	r.Handle("/metrics", promhttp.Handler())
```

- [ ] **Step 2: Run full backend test suite with race detector**

Run: `cd /home/sthomas/projects/ezyshare/backend && go build -o /tmp/ezyshare-server ./cmd/server && go test ./... -v -race -cover`
Expected: PASS across all packages with 0 data races.

- [ ] **Step 3: Commit**

```bash
git add backend/
git commit -m "feat(server): integrate IP rate limiting, CSP security headers, and MaxBytes payload guard"
```
