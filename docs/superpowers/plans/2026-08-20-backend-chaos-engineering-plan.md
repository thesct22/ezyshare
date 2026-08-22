# Header-Authenticated Production Chaos Engineering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement safe, header-authenticated Production Chaos Engineering in the Go backend to allow developers and automated test suites to simulate latency, dropped WebSocket signaling frames, and HTTP 500 errors via secret headers (`X-Chaos-Secret`) without affecting regular production users.

**Architecture:** Create `internal/middleware/chaos.go` and `chaos_test.go`. Update `internal/handler/ws.go` to support WebSocket chaos parameter checks. Integrate chaos middleware in `cmd/server/main.go`.

**Tech Stack:** Go standard library `crypto/subtle`, `net/http`, `time`, `math/rand`, `github.com/prometheus/client_golang`.

## Global Constraints

- Chaos MUST be 100% no-op for regular users who do not provide the secret `X-Chaos-Secret` header or query parameter.
- Must pass all tests with `go test -race ./...`.

---

### Task 1: Production Chaos Middleware & WebSocket Inspector (`internal/middleware/chaos.go` & `chaos_test.go`)

**Files:**
- Create: `backend/internal/middleware/chaos.go`
- Create: `backend/internal/middleware/chaos_test.go`

**Interfaces:**
- Produces:
  - `middleware.ChaosMiddleware(metrics *telemetry.Metrics) func(http.Handler) http.Handler`
  - `middleware.ShouldDropWebSocketFrame(r *http.Request) bool`
  - `middleware.GetChaosLatency(r *http.Request) time.Duration`

- [ ] **Step 1: Create `chaos.go`**

Create `backend/internal/middleware/chaos.go`:

```go
package middleware

import (
	"crypto/subtle"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func IsAuthorizedChaosRequest(r *http.Request) bool {
	prodSecret := os.Getenv("CHAOS_SECRET")
	if prodSecret == "" {
		return false
	}

	clientSecret := r.Header.Get("X-Chaos-Secret")
	if clientSecret == "" {
		clientSecret = r.URL.Query().Get("chaos_secret")
	}

	if clientSecret == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(clientSecret), []byte(prodSecret)) == 1
}

func GetChaosLatency(r *http.Request) time.Duration {
	if !IsAuthorizedChaosRequest(r) {
		return 0
	}

	latencyMsStr := r.Header.Get("X-Chaos-Latency-Ms")
	if latencyMsStr == "" {
		latencyMsStr = r.URL.Query().Get("chaos_latency_ms")
	}

	latencyMs, _ := strconv.Atoi(latencyMsStr)
	if latencyMs <= 0 {
		latencyMs = 300
	}

	return time.Duration(rand.Intn(latencyMs)) * time.Millisecond
}

func ShouldDropWebSocketFrame(r *http.Request) bool {
	if !IsAuthorizedChaosRequest(r) {
		return false
	}

	errorRateStr := r.Header.Get("X-Chaos-Error-Rate")
	if errorRateStr == "" {
		errorRateStr = r.URL.Query().Get("chaos_error_rate")
	}

	errorRate, _ := strconv.ParseFloat(errorRateStr, 64)
	if errorRate <= 0 {
		errorRate = 0.1
	}

	return rand.Float64() < errorRate
}

func ChaosMiddleware(metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsAuthorizedChaosRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Inject Artificial Jitter / Latency
			delay := GetChaosLatency(r)
			if delay > 0 {
				time.Sleep(delay)
			}

			// 2. Inject Artificial HTTP Fault
			errorRateStr := r.Header.Get("X-Chaos-Error-Rate")
			if errorRateStr == "" {
				errorRateStr = r.URL.Query().Get("chaos_error_rate")
			}
			errorRate, _ := strconv.ParseFloat(errorRateStr, 64)
			if errorRate <= 0 {
				errorRate = 0.1
			}

			if rand.Float64() < errorRate {
				slog.Warn("🔥 Authorized Production Chaos Injected",
					"client_ip", telemetry.GetClientIP(r),
					"path", r.URL.Path,
				)
				if metrics != nil && metrics.RateLimitExceeded != nil {
					metrics.RateLimitExceeded.WithLabelValues("chaos_injection").Inc()
				}
				http.Error(w, "500 Internal Server Error (Authorized Production Chaos Injected)", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 2: Create `chaos_test.go`**

Create `backend/internal/middleware/chaos_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/middleware"
)

func TestChaosMiddlewareUnauthorized(t *testing.T) {
	os.Setenv("CHAOS_SECRET", "my-secret-key")
	defer os.Unsetenv("CHAOS_SECRET")

	handler := middleware.ChaosMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unauthorized request without secret header, got %d", rec.Code)
	}
}

func TestChaosMiddlewareAuthorized(t *testing.T) {
	os.Setenv("CHAOS_SECRET", "my-secret-key")
	defer os.Unsetenv("CHAOS_SECRET")

	handler := middleware.ChaosMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	req.Header.Set("X-Chaos-Secret", "my-secret-key")
	req.Header.Set("X-Chaos-Error-Rate", "1.0") // 100% error rate for test
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error for authorized chaos request, got %d", rec.Code)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/middleware/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/middleware/
git commit -m "feat(middleware): add header-authenticated production chaos middleware"
```

---

### Task 2: Inject Chaos Check into WebSocket Handler (`internal/handler/ws.go`)

**Files:**
- Modify: `backend/internal/handler/ws.go`

- [ ] **Step 1: Update `ServeWS` in `ws.go`**

Update `backend/internal/handler/ws.go`:

```go
		var msg domain.SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			slog.Debug("WebSocket connection closed or read error", "error", err)
			break
		}

		// Chaos Fault Injection Hook for WebSocket signaling frames
		if middleware.IsAuthorizedChaosRequest(r) {
			delay := middleware.GetChaosLatency(r)
			if delay > 0 {
				time.Sleep(delay)
			}
			if middleware.ShouldDropWebSocketFrame(r) {
				slog.Warn("🔥 Production Chaos: Dropping WebSocket Signaling Frame", "peer_id", msg.SenderID, "type", msg.Type)
				continue
			}
		}
```

- [ ] **Step 2: Run handler tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v -race`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(handler): inject WebSocket signaling frame chaos hooks"
```

---

### Task 3: Server Integration & Comprehensive Verification (`cmd/server/main.go`)

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Mount `ChaosMiddleware` in `main.go`**

Update `backend/cmd/server/main.go`:

```go
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.SecurityHeaders(cfg.AppEnv))
	r.Use(customMiddleware.MaxBytesMiddleware(4 * 1024))
	r.Use(customMiddleware.ChaosMiddleware(metrics))
```

- [ ] **Step 2: Run full backend test suite with race detector**

Run: `cd /home/sthomas/projects/ezyshare/backend && go build -o /tmp/ezyshare-server ./cmd/server && go test ./... -v -race -cover`
Expected: PASS across all packages with 0 data races.

- [ ] **Step 3: Commit**

```bash
git add backend/
git commit -m "feat(server): integrate header-authenticated chaos middleware into Chi router"
```
