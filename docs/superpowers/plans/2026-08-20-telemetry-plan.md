# Telemetry & Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add structured `slog` logging (with custom client IP extraction) and Prometheus metrics instrumentation (counters, gauge, histogram) to the EzyShare Go backend.

**Architecture:** Create an `internal/telemetry` package for `slog` configuration, `GetClientIP` helper, and Prometheus metrics definitions (`active_peers` gauge, `messages_relayed_total` counter, `websocket_connections_total` counter, `http_requests_total` counter, `http_request_duration_seconds` histogram). Integrate telemetry into `signaling.Hub`, `handler.Handler`, and `cmd/server/main.go`.

**Tech Stack:** Go standard library `log/slog`, `github.com/prometheus/client_golang`, `github.com/go-chi/chi/v5`, `github.com/gorilla/websocket`.

## Global Constraints

- Use standard library `log/slog` for structured logging.
- Use `github.com/prometheus/client_golang/prometheus` and `promhttp` for metrics export.
- Implement custom `GetClientIP(r *http.Request) string` helper for IP resolution without deprecated `realip` middleware.

---

### Task 1: Create `internal/telemetry` Package (`logger.go` & `metrics.go`)

**Files:**
- Create: `backend/internal/telemetry/logger.go`
- Create: `backend/internal/telemetry/metrics.go`
- Create: `backend/internal/telemetry/metrics_test.go`
- Modify: `backend/go.mod`

**Interfaces:**
- Produces:
  - `telemetry.InitLogger(levelStr, format string) *slog.Logger`
  - `telemetry.GetClientIP(r *http.Request) string`
  - `telemetry.Metrics` struct with `ActivePeers` (Gauge), `MessagesRelayed` (CounterVec), `WebSocketConnections` (CounterVec), `HTTPRequestsTotal` (CounterVec), `HTTPRequestDuration` (HistogramVec)
  - `telemetry.NewMetrics(reg prometheus.Registerer) *Metrics`
  - `telemetry.HTTPMiddleware(metrics *Metrics) func(http.Handler) http.Handler`

- [ ] **Step 1: Write `logger.go`**

Create `backend/internal/telemetry/logger.go`:

```go
package telemetry

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
)

// InitLogger initializes standard library slog logger based on format and level strings.
func InitLogger(levelStr, format string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// GetClientIP resolves the client IP address from request headers or remote addr.
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if ip != "" {
			return ip
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
```

- [ ] **Step 2: Write `metrics.go`**

Create `backend/internal/telemetry/metrics.go`:

```go
package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	// Gauge metric
	ActivePeers prometheus.Gauge
	// Counter metrics
	MessagesRelayed      *prometheus.CounterVec
	WebSocketConnections *prometheus.CounterVec
	HTTPRequestsTotal    *prometheus.CounterVec
	// Histogram metric
	HTTPRequestDuration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ActivePeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ezyshare_active_peers",
			Help: "Current number of active WebRTC signaling peers connected.",
		}),
		MessagesRelayed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_messages_relayed_total",
				Help: "Total number of WebRTC signaling messages relayed.",
			},
			[]string{"type"},
		),
		WebSocketConnections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_websocket_connections_total",
				Help: "Total number of WebSocket connection events.",
			},
			[]string{"status"},
		),
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ezyshare_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}

	if reg != nil {
		reg.MustRegister(
			m.ActivePeers,
			m.MessagesRelayed,
			m.WebSocketConnections,
			m.HTTPRequestsTotal,
			m.HTTPRequestDuration,
		)
	}

	return m
}

func HTTPMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(ww.Status())
			path := r.URL.Path

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}
```

- [ ] **Step 3: Write tests for `telemetry`**

Create `backend/internal/telemetry/metrics_test.go`:

```go
package telemetry_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestMetricsRegistrationAndMiddleware(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)

	metrics.ActivePeers.Inc()
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 1 {
		t.Fatalf("expected ActivePeers 1, got %f", val)
	}

	handler := telemetry.HTTPMiddleware(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if count := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "200")); count != 1 {
		t.Fatalf("expected HTTPRequestsTotal 1, got %f", count)
	}
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	if ip := telemetry.GetClientIP(req); ip != "203.0.113.195" {
		t.Fatalf("expected X-Forwarded-For IP 203.0.113.195, got %s", ip)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Real-IP", "198.51.100.1")
	if ip := telemetry.GetClientIP(req2); ip != "198.51.100.1" {
		t.Fatalf("expected X-Real-IP 198.51.100.1, got %s", ip)
	}
}
```

- [ ] **Step 4: Run `go mod tidy` and run tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go mod tidy && go test ./internal/telemetry/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/telemetry backend/go.mod backend/go.sum
git commit -m "feat(telemetry): add slog logger with GetClientIP helper and prometheus metrics package"
```

---

### Task 2: Instrument `internal/signaling/hub.go` with Metrics

**Files:**
- Modify: `backend/internal/signaling/hub.go`
- Create: `backend/internal/signaling/hub_test.go`

**Interfaces:**
- Consumes: `telemetry.Metrics`
- Modifies: `signaling.NewHub(metrics *telemetry.Metrics) *Hub`

- [ ] **Step 1: Update `hub.go`**

Update `backend/internal/signaling/hub.go`:

```go
package signaling

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

var ErrPeerNotFound = errors.New("target peer not registered")

type Hub struct {
	mu       sync.RWMutex
	peers    map[string]domain.Client
	messages chan domain.SignalMessage
	quit     chan struct{}
	metrics  *telemetry.Metrics
}

func NewHub(metrics *telemetry.Metrics) *Hub {
	return &Hub{
		peers:    make(map[string]domain.Client),
		messages: make(chan domain.SignalMessage),
		quit:     make(chan struct{}),
		metrics:  metrics,
	}
}

func (h *Hub) Register(client domain.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[client.ID()] = client
	if h.metrics != nil {
		h.metrics.ActivePeers.Inc()
	}
	slog.Info("Peer registered in hub", "peer_id", client.ID())
}

func (h *Hub) Unregister(client domain.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.peers[client.ID()]; exists {
		delete(h.peers, client.ID())
		if h.metrics != nil {
			h.metrics.ActivePeers.Dec()
		}
		slog.Info("Peer unregistered from hub", "peer_id", client.ID())
	}
}

func (h *Hub) Relay(msg domain.SignalMessage) error {
	h.mu.RLock()
	target, ok := h.peers[msg.TargetID]
	h.mu.RUnlock()

	if !ok {
		slog.Warn("Target peer not found", "target_id", msg.TargetID)
		return ErrPeerNotFound
	}

	err := target.Send(msg)
	if err == nil && h.metrics != nil {
		h.metrics.MessagesRelayed.WithLabelValues(string(msg.Type)).Inc()
	}
	return err
}

func (h *Hub) Start() {
	slog.Info("Hub event loop started")
	for {
		select {
		case msg := <-h.messages:
			if err := h.Relay(msg); err != nil {
				slog.Error("Failed to relay message", "error", err)
			}
		case <-h.quit:
			slog.Info("Shutdown signal received. Closing all peer connections...")
			h.closeAllConnections()
			slog.Info("Hub event loop terminated")
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.quit)
}

func (h *Hub) closeAllConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, client := range h.peers {
		if err := client.Close(); err != nil {
			slog.Error("Failed to close peer connection", "peer_id", id, "error", err)
		}
		delete(h.peers, id)
		if h.metrics != nil {
			h.metrics.ActivePeers.Dec()
		}
	}
}
```

- [ ] **Step 2: Write tests for Hub metrics**

Create `backend/internal/signaling/hub_test.go`:

```go
package signaling_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type mockClient struct {
	id   string
	sent []domain.SignalMessage
}

func (m *mockClient) ID() string { return m.id }
func (m *mockClient) Send(msg domain.SignalMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}
func (m *mockClient) Close() error { return nil }

func TestHubMetricsInstrumentation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	c1 := &mockClient{id: "peer-1"}
	c2 := &mockClient{id: "peer-2"}

	hub.Register(c1)
	hub.Register(c2)

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 2 {
		t.Fatalf("expected 2 active peers, got %f", val)
	}

	msg := domain.SignalMessage{
		Type:     domain.TypeOffer,
		SenderID: "peer-1",
		TargetID: "peer-2",
	}

	if err := hub.Relay(msg); err != nil {
		t.Fatalf("unexpected relay error: %v", err)
	}

	if count := testutil.ToFloat64(metrics.MessagesRelayed.WithLabelValues("offer")); count != 1 {
		t.Fatalf("expected 1 offer message relayed, got %f", count)
	}

	hub.Unregister(c1)
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 1 {
		t.Fatalf("expected 1 active peer after unregister, got %f", val)
	}
}
```

- [ ] **Step 3: Run unit tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/signaling/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/signaling/
git commit -m "feat(signaling): instrument hub with active peers gauge and message relay counter metrics"
```

---

### Task 3: Instrument `internal/handler/ws.go` & Fix Disconnect Loop Cleanup

**Files:**
- Modify: `backend/internal/handler/ws.go`

**Interfaces:**
- Consumes: `signaling.Hub`, `telemetry.Metrics`
- Modifies: `handler.NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics) *Handler`

- [ ] **Step 1: Update `ws.go`**

Update `backend/internal/handler/ws.go`:

```go
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

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
	hub      *signaling.Hub
	metrics  *telemetry.Metrics
	upgrader websocket.Upgrader
}

func NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics) *Handler {
	return &Handler{
		hub:     hub,
		metrics: metrics,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
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

- [ ] **Step 2: Build verification**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/ws.go
git commit -m "feat(handler): add websocket counter metrics and client ip extraction"
```

---

### Task 4: Integrate Telemetry in `cmd/server/main.go` & End-to-End Verification

**Files:**
- Modify: `backend/cmd/server/main.go`

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

	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	logFormat := os.Getenv("LOG_FORMAT")
	logger := telemetry.InitLogger(logLevel, logFormat)

	metrics := telemetry.NewMetrics(prometheus.DefaultRegisterer)

	hub := signaling.NewHub(metrics)
	go hub.Start()

	wsHandler := handler.NewHandler(hub, metrics)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://localhost:*"},
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "addr", srv.Addr)
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

- [ ] **Step 2: Test building the binary**

Run: `cd /home/sthomas/projects/ezyshare/backend && go build -o /tmp/ezyshare-server ./cmd/server`
Expected: Zero build errors.

- [ ] **Step 3: Run all unit tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(server): integrate GetClientIP into slog middleware and expose /metrics endpoint"
```
