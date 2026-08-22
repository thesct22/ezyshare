# Backend Comprehensive Test Suite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build complete unit and integration tests across all backend packages (`telemetry`, `signaling`, `handler`, and `cmd/server`).

**Architecture:** Create modular tests using Go `testing` standard package, `prometheus/client_golang/prometheus/testutil`, `net/http/httptest`, and `gorilla/websocket` client dialer. Test concurrent safety, metrics recording, WebSocket upgrade & signaling relay flow, and server routes.

**Tech Stack:** Go `testing`, `httptest`, `prometheus/testutil`, `gorilla/websocket`.

## Global Constraints

- Run `go test -race ./...` to verify zero data races.
- Ensure all tests pass cleanly without hanging or leaving open goroutines.

---

### Task 1: Complete `internal/telemetry` Tests (`logger_test.go` & `metrics_test.go`)

**Files:**
- Create: `backend/internal/telemetry/logger_test.go`
- Modify: `backend/internal/telemetry/metrics_test.go`

**Interfaces:**
- Tests `telemetry.InitLogger`, `telemetry.GetClientIP`, `telemetry.NewMetrics`, `telemetry.HTTPMiddleware`

- [ ] **Step 1: Write `logger_test.go`**

Create `backend/internal/telemetry/logger_test.go`:

```go
package telemetry_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestInitLogger(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid"}
	formats := []string{"json", "text"}

	for _, lvl := range levels {
		for _, fmtStr := range formats {
			logger := telemetry.InitLogger(lvl, fmtStr)
			if logger == nil {
				t.Fatalf("expected non-nil logger for level %s, format %s", lvl, fmtStr)
			}
		}
	}
}

func TestGetClientIPCases(t *testing.T) {
	t.Run("X-Forwarded-For multiple IPs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		if ip := telemetry.GetClientIP(req); ip != "203.0.113.195" {
			t.Fatalf("expected 203.0.113.195, got %s", ip)
		}
	})

	t.Run("X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "198.51.100.1")
		if ip := telemetry.GetClientIP(req); ip != "198.51.100.1" {
			t.Fatalf("expected 198.51.100.1, got %s", ip)
		}
	})

	t.Run("RemoteAddr host port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.50:54321"
		if ip := telemetry.GetClientIP(req); ip != "192.168.1.50" {
			t.Fatalf("expected 192.168.1.50, got %s", ip)
		}
	})

	t.Run("RemoteAddr fallback without port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "invalid-address-format"
		if ip := telemetry.GetClientIP(req); ip != "invalid-address-format" {
			t.Fatalf("expected fallback invalid-address-format, got %s", ip)
		}
	})
}
```

- [ ] **Step 2: Update `metrics_test.go`**

Update `backend/internal/telemetry/metrics_test.go`:

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
	metrics.ActivePeers.Dec()
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected ActivePeers 0, got %f", val)
	}

	metrics.MessagesRelayed.WithLabelValues("offer").Inc()
	metrics.MessagesRelayed.WithLabelValues("answer").Inc()
	if count := testutil.ToFloat64(metrics.MessagesRelayed.WithLabelValues("offer")); count != 1 {
		t.Fatalf("expected MessagesRelayed offer 1, got %f", count)
	}

	metrics.WebSocketConnections.WithLabelValues("connected").Inc()
	metrics.WebSocketConnections.WithLabelValues("disconnected").Inc()
	if count := testutil.ToFloat64(metrics.WebSocketConnections.WithLabelValues("connected")); count != 1 {
		t.Fatalf("expected WebSocketConnections connected 1, got %f", count)
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

- [ ] **Step 3: Run telemetry tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/telemetry/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/telemetry/
git commit -m "test(telemetry): add complete test coverage for logger and metrics"
```

---

### Task 2: Comprehensive `internal/signaling` Tests (`hub_test.go`)

**Files:**
- Modify: `backend/internal/signaling/hub_test.go`

**Interfaces:**
- Tests `signaling.Hub.Register`, `Unregister`, `Relay`, `Start`, `Stop`, `closeAllConnections`

- [ ] **Step 1: Expand `hub_test.go`**

Update `backend/internal/signaling/hub_test.go`:

```go
package signaling_test

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type mockClient struct {
	mu     sync.Mutex
	id     string
	sent   []domain.SignalMessage
	closed bool
}

func (m *mockClient) ID() string { return m.id }

func (m *mockClient) Send(msg domain.SignalMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockClient) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestHubRegisterUnregisterConcurrent(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c := &mockClient{id: string(rune('A' + id))}
			hub.Register(c)
			hub.Unregister(c)
		}(i)
	}
	wg.Wait()

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected 0 active peers after concurrent register/unregister, got %f", val)
	}
}

func TestHubRelayErrPeerNotFound(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	msg := domain.SignalMessage{
		Type:     domain.TypeOffer,
		SenderID: "peer-1",
		TargetID: "non-existent-peer",
	}

	err := hub.Relay(msg)
	if err != signaling.ErrPeerNotFound {
		t.Fatalf("expected ErrPeerNotFound, got %v", err)
	}
}

func TestHubLifecycleAndStop(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	c1 := &mockClient{id: "peer-1"}
	c2 := &mockClient{id: "peer-2"}
	hub.Register(c1)
	hub.Register(c2)

	go hub.Start()
	time.Sleep(10 * time.Millisecond)

	hub.Stop()
	time.Sleep(10 * time.Millisecond)

	if !c1.isClosed() || !c2.isClosed() {
		t.Fatalf("expected all clients to be closed on hub stop")
	}

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected 0 active peers after hub stop, got %f", val)
	}
}
```

- [ ] **Step 2: Run hub tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/signaling/... -v -race`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/signaling/hub_test.go
git commit -m "test(signaling): add concurrent registration, relay error, and lifecycle hub tests"
```

---

### Task 3: Comprehensive `internal/handler` Tests (`ws_test.go`)

**Files:**
- Create: `backend/internal/handler/ws_test.go`

**Interfaces:**
- Tests `handler.Handler.ServeWS`, WebSocket upgrade, join, WebRTC offer/answer/candidate relay, unauthenticated frames, and disconnection.

- [ ] **Step 1: Create `ws_test.go`**

Create `backend/internal/handler/ws_test.go`:

```go
package handler_test

import (
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

	wsHandler := handler.NewHandler(hub, metrics)

	ts := httptest.NewServer(wsHandler.ServeWS)
	return ts, hub, metrics
}

func dialWS(url string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	return conn, err
}

func TestWebSocketSignalingExchange(t *testing.T) {
	ts, hub, metrics := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	// Dial Client A
	connA, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client A: %v", err)
	}
	defer connA.Close()

	// Dial Client B
	connB, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client B: %v", err)
	}
	defer connB.Close()

	// Client A joins
	joinA := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-a"}
	if err := connA.WriteJSON(joinA); err != nil {
		t.Fatalf("Client A join write failed: %v", err)
	}

	// Client B joins
	joinB := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-b"}
	if err := connB.WriteJSON(joinB); err != nil {
		t.Fatalf("Client B join write failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 2 {
		t.Fatalf("expected 2 active peers in hub, got %f", val)
	}

	// Client A sends offer to Client B
	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "client-b",
		Payload:  "offer-sdp-data",
	}
	if err := connA.WriteJSON(offer); err != nil {
		t.Fatalf("Client A offer write failed: %v", err)
	}

	// Client B reads offer
	var receivedOffer domain.SignalMessage
	connB.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connB.ReadJSON(&receivedOffer); err != nil {
		t.Fatalf("Client B read offer failed: %v", err)
	}

	if receivedOffer.Type != domain.TypeOffer || receivedOffer.SenderID != "client-a" {
		t.Fatalf("Client B received invalid offer: %+v", receivedOffer)
	}

	// Client B sends answer to Client A
	answer := domain.SignalMessage{
		Type:     domain.TypeAnswer,
		TargetID: "client-a",
		Payload:  "answer-sdp-data",
	}
	if err := connB.WriteJSON(answer); err != nil {
		t.Fatalf("Client B answer write failed: %v", err)
	}

	// Client A reads answer
	var receivedAnswer domain.SignalMessage
	connA.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connA.ReadJSON(&receivedAnswer); err != nil {
		t.Fatalf("Client A read answer failed: %v", err)
	}

	if receivedAnswer.Type != domain.TypeOffer && receivedAnswer.Type != domain.TypeAnswer || receivedAnswer.SenderID != "client-b" {
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

	// Send offer without join
	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "some-target",
	}
	if err := conn.WriteJSON(offer); err != nil {
		t.Fatalf("write offer failed: %v", err)
	}

	// Connection stays alive, send join now
	join := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-unauth"}
	if err := conn.WriteJSON(join); err != nil {
		t.Fatalf("write join failed: %v", err)
	}
}
```

- [ ] **Step 2: Run handler tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v -race`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/ws_test.go
git commit -m "test(handler): add full websocket upgrade, P2P signaling exchange, and unauthenticated frame tests"
```

---

### Task 4: Integration Tests for `cmd/server` (`server_test.go`) & Final Suite Verification

**Files:**
- Create: `backend/cmd/server/server_test.go`

**Interfaces:**
- Tests `GET /healthz`, `GET /metrics`, CORS `OPTIONS /ws`

- [ ] **Step 1: Create `server_test.go`**

Create `backend/cmd/server/server_test.go`:

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
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func setupTestRouter() http.Handler {
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
		AllowCredentials: true,
	}))
	r.Use(telemetry.HTTPMiddleware(metrics))

	r.Get("/ws", wsHandler.ServeWS)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Handle("/metrics", promhttp.Handler())

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
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 CORS status, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run all backend tests with race detection and coverage**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./... -v -race -cover`
Expected: PASS across all packages.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/server_test.go
git commit -m "test(server): add integration tests for healthz, metrics, and CORS endpoints"
```
