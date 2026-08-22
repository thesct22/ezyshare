# Security & Environment Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add environment-based configuration switching (`dev`, `staging`, `prod`), strict CORS origin enforcement (`https://sharath.is-a.dev` in production), WebSocket origin validation, HTTP security headers, and WebSocket frame size limits.

**Architecture:** Create an `internal/config` package to manage environment settings and origin validation. Update `handler.Handler` to enforce origins and read limits. Update `cmd/server/main.go` to inject config and security headers.

**Tech Stack:** Go standard library `os`, `net/url`, `github.com/go-chi/cors`, `github.com/gorilla/websocket`.

## Global Constraints

- Default production CORS allowed origin is `https://sharath.is-a.dev`.
- All tests must pass with `go test -race ./...`.

---

### Task 1: Create `internal/config` Package (`config.go` & `config_test.go`)

**Files:**

- Create: `backend/internal/config/config.go`
- Create: `backend/internal/config/config_test.go`

**Interfaces:**

- Produces:
  - `config.Config` struct (`AppEnv`, `Port`, `LogLevel`, `LogFormat`, `AllowedOrigins`)
  - `config.LoadConfig() *Config`
  - `config.IsOriginAllowed(origin string, allowedOrigins []string) bool`

- [ ] **Step 1: Write `config.go`**

Create `backend/internal/config/config.go`:

```go
package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppEnv         string
	Port           string
	LogLevel       string
	LogFormat      string
	AllowedOrigins []string
}

func LoadConfig() *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "dev"
	}
	env = strings.ToLower(env)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		if env == "dev" {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		if env == "dev" {
			logFormat = "text"
		} else {
			logFormat = "json"
		}
	}

	var origins []string
	if custom := os.Getenv("ALLOWED_ORIGINS"); custom != "" {
		for _, o := range strings.Split(custom, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	} else {
		switch env {
		case "prod":
			origins = []string{"https://sharath.is-a.dev"}
		case "staging":
			origins = []string{"https://sharath.is-a.dev", "http://localhost:*", "http://127.0.0.1:*"}
		default: // dev
			origins = []string{"http://localhost:*", "http://127.0.0.1:*", "https://sharath.is-a.dev"}
		}
	}

	return &Config{
		AppEnv:         env,
		Port:           port,
		LogLevel:       logLevel,
		LogFormat:      logFormat,
		AllowedOrigins: origins,
	}
}

// IsOriginAllowed checks if an incoming origin matches the allowed origin patterns.
func IsOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	requestHost := u.Host
	requestScheme := u.Scheme

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}

		allowedURL, err := url.Parse(allowed)
		if err != nil {
			continue
		}

		if allowedURL.Scheme != "" && allowedURL.Scheme != requestScheme {
			continue
		}

		allowedHost := allowedURL.Host
		if allowedHost == "" {
			allowedHost = allowed
		}

		if matched, _ := filepath.Match(allowedHost, requestHost); matched {
			return true
		}
		if allowedHost == requestHost {
			return true
		}
	}

	return false
}
```

- [ ] **Step 2: Write `config_test.go`**

Create `backend/internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	os.Unsetenv("APP_ENV")
	os.Unsetenv("ENV")
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg := config.LoadConfig()
	if cfg.AppEnv != "dev" {
		t.Fatalf("expected AppEnv dev, got %s", cfg.AppEnv)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected Port 8080, got %s", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Fatalf("expected LogFormat text, got %s", cfg.LogFormat)
	}
}

func TestLoadConfigProd(t *testing.T) {
	os.Setenv("APP_ENV", "prod")
	defer os.Unsetenv("APP_ENV")

	cfg := config.LoadConfig()
	if cfg.AppEnv != "prod" {
		t.Fatalf("expected AppEnv prod, got %s", cfg.AppEnv)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected LogLevel info, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("expected LogFormat json, got %s", cfg.LogFormat)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "https://sharath.is-a.dev" {
		t.Fatalf("expected prod allowed origin https://sharath.is-a.dev, got %v", cfg.AllowedOrigins)
	}
}

func TestIsOriginAllowed(t *testing.T) {
	origins := []string{"https://sharath.is-a.dev", "http://localhost:*"}

	if !config.IsOriginAllowed("https://sharath.is-a.dev", origins) {
		t.Fatalf("expected https://sharath.is-a.dev to be allowed")
	}
	if !config.IsOriginAllowed("http://localhost:3000", origins) {
		t.Fatalf("expected http://localhost:3000 to be allowed")
	}
	if config.IsOriginAllowed("https://malicious-website.com", origins) {
		t.Fatalf("expected https://malicious-website.com to be rejected")
	}
}
```

- [ ] **Step 3: Run config tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/config/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/
git commit -m "feat(config): add environment configuration and origin validation package"
```

---

### Task 2: Update `internal/handler/ws.go` with Strict Origin Validation & Payload Limits

**Files:**

- Modify: `backend/internal/handler/ws.go`
- Modify: `backend/internal/handler/ws_test.go`

**Interfaces:**

- Consumes: `config.IsOriginAllowed`
- Modifies: `handler.NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics, allowedOrigins []string) *Handler`

- [ ] **Step 1: Update `ws.go`**

Update `backend/internal/handler/ws.go`:

```go
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type wsClient struct {
	id   string
	conn *websocket.Conn
}

func (c *wsClient) ID() string {
	return c.id
}

func (c *wsClient) Send(msg domain.SignalMessage) error {
	return c.conn.WriteJSON(msg)
}

func (c *wsClient) Close() error {
	return c.conn.Close()
}

type Handler struct {
	hub            *signaling.Hub
	metrics        *telemetry.Metrics
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

func NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics, allowedOrigins []string) *Handler {
	h := &Handler{
		hub:            hub,
		metrics:        metrics,
		allowedOrigins: allowedOrigins,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Allow requests without Origin header (e.g. CLI or mobile apps)
				return true
			}
			return config.IsOriginAllowed(origin, h.allowedOrigins)
		},
	}

	return h
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err, "client_ip", telemetry.GetClientIP(r))
		if h.metrics != nil {
			h.metrics.WebSocketConnections.WithLabelValues("failed").Inc()
		}
		return
	}
	defer conn.Close()

	// Enforce max 64KB per read frame to protect server memory
	conn.SetReadLimit(64 * 1024)

	if h.metrics != nil {
		h.metrics.WebSocketConnections.WithLabelValues("connected").Inc()
	}

	var client *wsClient

	defer func() {
		if client != nil {
			h.hub.Unregister(client)
			slog.Info("Peer disconnected", "peer_id", client.ID())
		}
		if h.metrics != nil {
			h.metrics.WebSocketConnections.WithLabelValues("disconnected").Inc()
		}
	}()

	for {
		var msg domain.SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			slog.Debug("WebSocket connection closed or read error", "error", err)
			break
		}

		switch msg.Type {
		case domain.TypeJoin:
			if msg.SenderID == "" {
				slog.Warn("Join attempt missing senderId")
				continue
			}
			client = &wsClient{
				id:   msg.SenderID,
				conn: conn,
			}
			h.hub.Register(client)

		case domain.TypeLeave:
			if client != nil {
				slog.Info("Peer requested leave", "peer_id", client.ID())
			}
			return

		case domain.TypeOffer, domain.TypeAnswer, domain.TypeCandidate:
			if client == nil {
				slog.Warn("Unauthenticated signaling frame received before join", "type", msg.Type)
				continue
			}

			msg.SenderID = client.ID()
			if err := h.hub.Relay(msg); err != nil {
				if errors.Is(err, signaling.ErrPeerNotFound) {
					slog.Debug("Failed to relay message: target not found", "targetId", msg.TargetID)
				} else {
					slog.Error("Error relaying signal message", "error", err, "targetId", msg.TargetID)
				}
			}

		default:
			slog.Warn("Unknown message type", "type", msg.Type)
		}
	}
}
```

- [ ] **Step 2: Update `ws_test.go`**

Update `backend/internal/handler/ws_test.go` to pass `[]string{"*"}` in `setupTestServer` and add `TestWebSocketOriginValidation`.

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func setupTestServer() (*httptest.Server, *signaling.Hub, *telemetry.Metrics) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()

	wsHandler := handler.NewHandler(hub, metrics, []string{"*"})

	ts := httptest.NewServer(http.HandlerFunc(wsHandler.ServeWS))
	return ts, hub, metrics
}

func dialWS(url string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	return conn, err
}

func TestWebSocketOriginValidation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()
	defer hub.Stop()

	wsHandler := handler.NewHandler(hub, metrics, []string{"https://sharath.is-a.dev"})
	ts := httptest.NewServer(http.HandlerFunc(wsHandler.ServeWS))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Dial with unauthorized origin
	headerFail := make(http.Header)
	headerFail.Set("Origin", "https://unauthorized-site.com")
	_, respFail, errFail := websocket.DefaultDialer.Dial(wsURL, headerFail)
	if errFail == nil || respFail == nil || respFail.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthorized origin")
	}

	// Dial with authorized origin
	headerSuccess := make(http.Header)
	headerSuccess.Set("Origin", "https://sharath.is-a.dev")
	connSuccess, _, errSuccess := websocket.DefaultDialer.Dial(wsURL, headerSuccess)
	if errSuccess != nil {
		t.Fatalf("expected origin https://sharath.is-a.dev to succeed: %v", errSuccess)
	}
	connSuccess.Close()
}

func TestWebSocketSignalingExchange(t *testing.T) {
	ts, hub, metrics := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	connA, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client A: %v", err)
	}
	defer connA.Close()

	connB, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client B: %v", err)
	}
	defer connB.Close()

	joinA := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-a"}
	if err := connA.WriteJSON(joinA); err != nil {
		t.Fatalf("Client A join write failed: %v", err)
	}

	joinB := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-b"}
	if err := connB.WriteJSON(joinB); err != nil {
		t.Fatalf("Client B join write failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 2 {
		t.Fatalf("expected 2 active peers in hub, got %f", val)
	}

	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "client-b",
		Payload:  "offer-sdp-data",
	}
	if err := connA.WriteJSON(offer); err != nil {
		t.Fatalf("Client A offer write failed: %v", err)
	}

	var receivedOffer domain.SignalMessage
	connB.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connB.ReadJSON(&receivedOffer); err != nil {
		t.Fatalf("Client B read offer failed: %v", err)
	}

	if receivedOffer.Type != domain.TypeOffer || receivedOffer.SenderID != "client-a" {
		t.Fatalf("Client B received invalid offer: %+v", receivedOffer)
	}

	answer := domain.SignalMessage{
		Type:     domain.TypeAnswer,
		TargetID: "client-a",
		Payload:  "answer-sdp-data",
	}
	if err := connB.WriteJSON(answer); err != nil {
		t.Fatalf("Client B answer write failed: %v", err)
	}

	var receivedAnswer domain.SignalMessage
	connA.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connA.ReadJSON(&receivedAnswer); err != nil {
		t.Fatalf("Client A read answer failed: %v", err)
	}

	if receivedAnswer.Type != domain.TypeAnswer || receivedAnswer.SenderID != "client-b" {
		t.Fatalf("Client A received invalid answer: %+v", receivedAnswer)
	}
}

func TestUnauthenticatedSignalingFrame(t *testing.T) {
	ts, hub, _ := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	conn, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial WS: %v", err)
	}
	defer conn.Close()

	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "some-target",
	}
	if err := conn.WriteJSON(offer); err != nil {
		t.Fatalf("write offer failed: %v", err)
	}

	join := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-unauth"}
	if err := conn.WriteJSON(join); err != nil {
		t.Fatalf("write join failed: %v", err)
	}
}
```

- [ ] **Step 3: Run handler tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(handler): add strict websocket origin check and 64KB read frame limit"
```

---

### Task 3: Integrate Security Headers & Config in `cmd/server/main.go` and Update Integration Tests

**Files:**

- Modify: `backend/cmd/server/main.go`
- Modify: `backend/cmd/server/server_test.go`

- [ ] **Step 1: Update `main.go`**

Update `backend/cmd/server/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func main() {
	cfg := config.LoadConfig()

	_ = telemetry.InitLogger(cfg.LogLevel, cfg.LogFormat)

	slog.Info("Configuration loaded",
		"env", cfg.AppEnv,
		"port", cfg.Port,
		"allowed_origins", cfg.AllowedOrigins,
	)

	metrics := telemetry.NewMetrics(prometheus.DefaultRegisterer)

	hub := signaling.NewHub(metrics)
	go hub.Start()

	wsHandler := handler.NewHandler(hub, metrics, cfg.AllowedOrigins)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Security Headers Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(telemetry.HTTPMiddleware(metrics))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start).String(),
				"client_ip", telemetry.GetClientIP(r),
				"req_id", middleware.GetReqID(r.Context()),
			)
		})
	})

	r.Get("/ws", wsHandler.ServeWS)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	slog.Info("Received shutdown signal. Stopping services...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	hub.Stop()

	slog.Info("Server shutdown completed cleanly.")
}
```

- [ ] **Step 2: Update `server_test.go`**

Update `backend/cmd/server/server_test.go`:

```go
package main_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func setupTestRouter() http.Handler {
	cfg := config.LoadConfig()
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()
	wsHandler := handler.NewHandler(hub, metrics, cfg.AllowedOrigins)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))
	r.Use(telemetry.HTTPMiddleware(metrics))

	r.Get("/ws", wsHandler.ServeWS)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	return r
}

func TestHealthzEndpoint(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options header")
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "OK" {
		t.Fatalf("expected body OK, got %s", string(body))
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "ezyshare_active_peers") {
		t.Fatalf("expected metrics response to contain ezyshare_active_peers")
	}
}

func TestCORSPreflight(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("OPTIONS", "/ws", nil)
	req.Header.Set("Origin", "https://sharath.is-a.dev")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 CORS status, got %d", rec.Code)
	}
}
```

- [ ] **Step 3: Test building binary & run all tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go build -o /tmp/ezyshare-server ./cmd/server && go test ./... -v -race -cover`
Expected: PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/
git commit -m "feat(server): integrate config package, security headers, and CORS allowed origins"
```
