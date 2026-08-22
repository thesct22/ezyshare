# Backend Zero-Knowledge Room Signaling & ICE Credentials Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the Go backend to support in-memory Zero-Knowledge Room Management (`room_id` custom UIDs / system UUIDs), room signaling, ephemeral TURN ICE credentials (`GET /api/v1/ice-servers`), and Prometheus room metrics without storing file metadata or passwords.

**Architecture:** Create an `internal/signaling/room.go` & `room_manager.go` package for room routing and auto-cleanup. Add an ICE servers handler for STUN/TURN credentials. Update WebSocket handlers and Chi server routes.

**Tech Stack:** Go standard library `sync`, `crypto/hmac`, `crypto/sha1`, `github.com/prometheus/client_golang`, `github.com/go-chi/chi/v5`.

## Global Constraints

- Backend MUST NOT store file names, file sizes, file contents, or password hashes.
- Backend MUST operate as a blind room signaling broker.
- All tests must pass with `go test -race ./...`.

---

### Task 1: Domain & Signaling Protocols Expansion (`signal.go` & `room.go`)

**Files:**

- Modify: `backend/internal/domain/signal.go`
- Create: `backend/internal/domain/room.go`

**Interfaces:**

- Produces:
  - `domain.MessageType` constants (`TypeCreateRoom`, `TypeJoinRoom`, `TypeLeaveRoom`, `TypePeerJoined`, `TypePeerLeft`, `TypeRoomCreated`, `TypeError`)
  - `domain.Room` struct (`ID`, `HostID`, `Peers map[string]Client`, `CreatedAt`, `LastActive`)

- [ ] **Step 1: Update `signal.go`**

Update `backend/internal/domain/signal.go`:

```go
package domain

// MessageType defines custom string types for signal protocol actions
type MessageType string

const (
	TypeJoin        MessageType = "join"
	TypeLeave       MessageType = "leave"
	TypeOffer       MessageType = "offer"
	TypeAnswer      MessageType = "answer"
	TypeCandidate   MessageType = "candidate"
	TypeCreateRoom  MessageType = "create_room"
	TypeJoinRoom    MessageType = "join_room"
	TypeLeaveRoom   MessageType = "leave_room"
	TypePeerJoined  MessageType = "peer_joined"
	TypePeerLeft    MessageType = "peer_left"
	TypeRoomCreated MessageType = "room_created"
	TypeError       MessageType = "error"
)

// SignalMessage is the struct representing JSON payloads exchanged over WebSockets.
type SignalMessage struct {
	Type     MessageType `json:"type"`
	SenderID string      `json:"sender_id"`
	TargetID string      `json:"target_id,omitempty"`
	RoomID   string      `json:"room_id,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

// Client is an interface that abstracts a connected peer connection.
type Client interface {
	ID() string
	Send(msg SignalMessage) error
	Close() error
}
```

- [ ] **Step 2: Create `room.go`**

Create `backend/internal/domain/room.go`:

```go
package domain

import (
	"sync"
	"time"
)

// Room represents a transient in-memory zero-knowledge signaling room.
type Room struct {
	mu         sync.RWMutex
	ID         string
	HostID     string
	Peers      map[string]Client
	CreatedAt  time.Time
	LastActive time.Time
}

func NewRoom(id, hostID string) *Room {
	now := time.Now()
	return &Room{
		ID:         id,
		HostID:     hostID,
		Peers:      make(map[string]Client),
		CreatedAt:  now,
		LastActive: now,
	}
}

func (r *Room) AddPeer(client Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Peers[client.ID()] = client
	r.LastActive = time.Now()
}

func (r *Room) RemovePeer(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Peers, clientID)
	r.LastActive = time.Now()
}

func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Peers)
}

func (r *Room) Broadcast(msg SignalMessage, excludeClientID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, client := range r.Peers {
		if id != excludeClientID {
			_ = client.Send(msg)
		}
	}
}
```

- [ ] **Step 3: Run domain tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/domain/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/
git commit -m "feat(domain): add room struct and room signaling message types"
```

---

### Task 2: Implement RoomManager (`room_manager.go` & `room_manager_test.go`)

**Files:**

- Create: `backend/internal/signaling/room_manager.go`
- Create: `backend/internal/signaling/room_manager_test.go`
- Modify: `backend/internal/telemetry/metrics.go`

**Interfaces:**

- Produces:
  - `signaling.RoomManager` (`GetOrCreateRoom`, `JoinRoom`, `LeaveRoom`, `RelayRoomSignal`)
  - Prometheus metrics `ActiveRooms` Gauge and `RoomsCreatedTotal` CounterVec

- [ ] **Step 1: Update `telemetry/metrics.go`**

Update `backend/internal/telemetry/metrics.go` to add `ActiveRooms` and `RoomsCreatedTotal`:

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
	// Histogram metric
	HTTPRequestDuration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ActivePeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ezyshare_active_peers",
			Help: "Current number of active WebRTC signaling peers connected.",
		}),
		ActiveRooms: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ezyshare_active_rooms",
			Help: "Current number of active transient signaling rooms.",
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
		RoomsCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_rooms_created_total",
				Help: "Total number of rooms created.",
			},
			[]string{"type"}, // custom_uid vs uuid
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
			m.ActiveRooms,
			m.MessagesRelayed,
			m.WebSocketConnections,
			m.RoomsCreatedTotal,
			m.HTTPRequestsTotal,
			m.HTTPRequestDuration,
		)
	}

	return m
}
```

- [ ] **Step 2: Create `room_manager.go`**

Create `backend/internal/signaling/room_manager.go`:

```go
package signaling

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomIDTaken   = errors.New("custom room ID already in use")
	ErrRoomFull      = errors.New("room reached maximum peer capacity")
)

const MaxPeersPerRoom = 10

type RoomManager struct {
	mu      sync.RWMutex
	rooms   map[string]*domain.Room
	metrics *telemetry.Metrics
	quit    chan struct{}
}

func NewRoomManager(metrics *telemetry.Metrics) *RoomManager {
	rm := &RoomManager{
		rooms:   make(map[string]*domain.Room),
		metrics: metrics,
		quit:    make(chan struct{}),
	}
	go rm.startCleanupLoop()
	return rm
}

func (rm *RoomManager) CreateRoom(customID, hostID string) (*domain.Room, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	roomID := customID
	isCustom := true
	if roomID == "" {
		roomID = rm.generateUUID()
		isCustom = false
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

func (rm *RoomManager) JoinRoom(roomID string, client domain.Client) (*domain.Room, error) {
	rm.mu.Lock()
	room, exists := rm.rooms[roomID]
	rm.mu.Unlock()

	if !exists {
		return nil, ErrRoomNotFound
	}

	if room.PeerCount() >= MaxPeersPerRoom {
		return nil, ErrRoomFull
	}

	room.AddPeer(client)
	slog.Info("Peer joined room", "room_id", roomID, "peer_id", client.ID())

	// Notify existing peers
	room.Broadcast(domain.SignalMessage{
		Type:     domain.TypePeerJoined,
		SenderID: client.ID(),
		RoomID:   roomID,
	}, client.ID())

	return room, nil
}

func (rm *RoomManager) LeaveRoom(roomID string, clientID string) {
	rm.mu.Lock()
	room, exists := rm.rooms[roomID]
	rm.mu.Unlock()

	if !exists {
		return
	}

	room.RemovePeer(clientID)
	slog.Info("Peer left room", "room_id", roomID, "peer_id", clientID)

	room.Broadcast(domain.SignalMessage{
		Type:     domain.TypePeerLeft,
		SenderID: clientID,
		RoomID:   roomID,
	}, clientID)

	if room.PeerCount() == 0 {
		rm.mu.Lock()
		delete(rm.rooms, roomID)
		if rm.metrics != nil {
			rm.metrics.ActiveRooms.Dec()
		}
		rm.mu.Unlock()
		slog.Info("Empty room destroyed", "room_id", roomID)
	}
}

func (rm *RoomManager) RelayRoomSignal(roomID string, msg domain.SignalMessage) error {
	rm.mu.RLock()
	room, exists := rm.rooms[roomID]
	rm.mu.RUnlock()

	if !exists {
		return ErrRoomNotFound
	}

	if msg.TargetID != "" {
		room.mu.RLock()
		target, ok := room.Peers[msg.TargetID]
		room.mu.RUnlock()

		if !ok {
			return ErrPeerNotFound
		}
		return target.Send(msg)
	}

	// Broadcast to all room peers except sender
	room.Broadcast(msg, msg.SenderID)
	return nil
}

func (rm *RoomManager) Stop() {
	close(rm.quit)
}

func (rm *RoomManager) startCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.cleanInactiveRooms()
		case <-rm.quit:
			return
		}
	}
}

func (rm *RoomManager) cleanInactiveRooms() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	for id, room := range rm.rooms {
		if room.PeerCount() == 0 || now.Sub(room.LastActive) > 1*time.Hour {
			delete(rm.rooms, id)
			if rm.metrics != nil {
				rm.metrics.ActiveRooms.Dec()
			}
			slog.Info("Cleaned inactive room", "room_id", id)
		}
	}
}

func (rm *RoomManager) generateUUID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("room-%s", string(b))
}
```

- [ ] **Step 3: Create `room_manager_test.go`**

Create `backend/internal/signaling/room_manager_test.go`:

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

func TestRoomManagerLifecycle(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	rm := signaling.NewRoomManager(metrics)
	defer rm.Stop()

	c1 := &mockClient{id: "peer-1"}
	c2 := &mockClient{id: "peer-2"}

	// Create custom room
	room, err := rm.CreateRoom("my-custom-room", c1.ID())
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}

	if val := testutil.ToFloat64(metrics.ActiveRooms); val != 1 {
		t.Fatalf("expected 1 active room, got %f", val)
	}

	// Duplicate custom room ID fails
	_, errDup := rm.CreateRoom("my-custom-room", "peer-3")
	if errDup != signaling.ErrRoomIDTaken {
		t.Fatalf("expected ErrRoomIDTaken, got %v", errDup)
	}

	// Join room
	_, errJoin := rm.JoinRoom("my-custom-room", c2)
	if errJoin != nil {
		t.Fatalf("failed to join room: %v", errJoin)
	}

	if room.PeerCount() != 1 { // c2 joined (c1 creates but joins via JoinRoom)
		t.Fatalf("expected 1 peer in room, got %d", room.PeerCount())
	}

	// Leave room
	rm.LeaveRoom("my-custom-room", c2.ID())
	if val := testutil.ToFloat64(metrics.ActiveRooms); val != 0 {
		t.Fatalf("expected 0 active rooms after empty cleanup, got %f", val)
	}
}
```

- [ ] **Step 4: Run signaling & telemetry tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/signaling/... ./internal/telemetry/... -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/signaling/ backend/internal/telemetry/
git commit -m "feat(signaling): implement zero-knowledge RoomManager and active rooms prometheus metrics"
```

---

### Task 3: Ephemeral TURN ICE Credentials Handler (`GET /api/v1/ice-servers`)

**Files:**

- Create: `backend/internal/handler/ice.go`
- Create: `backend/internal/handler/ice_test.go`

**Interfaces:**

- Produces:
  - `handler.ICEServerHandler(w http.ResponseWriter, r *http.Request)`
  - Endpoint `GET /api/v1/ice-servers` returning STUN & ephemeral Coturn TURN credentials

- [ ] **Step 1: Create `ice.go`**

Create `backend/internal/handler/ice.go`:

```go
package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type ICEResponse struct {
	ICEServers []ICEServer `json:"iceServers"`
}

func HandleICEServers(w http.ResponseWriter, r *http.Request) {
	turnServerURL := os.Getenv("TURN_SERVER_URL")
	turnSecret := os.Getenv("TURN_SECRET")

	iceServers := []ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
		},
	}

	if turnServerURL != "" && turnSecret != "" {
		// Generate ephemeral Coturn HMAC-SHA1 credentials (valid for 24 hours)
		ttl := 24 * time.Hour
		timestamp := time.Now().Add(ttl).Unix()
		username := fmt.Sprintf("%d:ezyshare-user", timestamp)

		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(username))
		credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		iceServers = append(iceServers, ICEServer{
			URLs:       []string{turnServerURL},
			Username:   username,
			Credential: credential,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ICEResponse{ICEServers: iceServers})
}
```

- [ ] **Step 2: Create `ice_test.go`**

Create `backend/internal/handler/ice_test.go`:

```go
package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/handler"
)

func TestHandleICEServers(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.HandleICEServers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp handler.ICEResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(resp.ICEServers) == 0 {
		t.Fatalf("expected at least 1 STUN server in response")
	}
}

func TestHandleICEServersWithTURN(t *testing.T) {
	os.Setenv("TURN_SERVER_URL", "turn:turn.ezyshare.dev:3478")
	os.Setenv("TURN_SECRET", "my-secret-key")
	defer os.Unsetenv("TURN_SERVER_URL")
	defer os.Unsetenv("TURN_SECRET")

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.HandleICEServers(rec, req)

	var resp handler.ICEResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.ICEServers) < 2 {
		t.Fatalf("expected STUN + TURN servers, got %d", len(resp.ICEServers))
	}
	if resp.ICEServers[1].Username == "" || resp.ICEServers[1].Credential == "" {
		t.Fatalf("expected non-empty TURN credentials")
	}
}
```

- [ ] **Step 3: Run handler tests**

Run: `cd /home/sthomas/projects/ezyshare/backend && go test ./internal/handler/... -v -race`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat(handler): add ephemeral TURN ICE credentials endpoint GET /api/v1/ice-servers"
```

---

### Task 4: Integrate RoomManager & ICE Servers Route in `cmd/server/main.go`

**Files:**

- Modify: `backend/cmd/server/main.go`
- Modify: `backend/cmd/server/server_test.go`
- Modify: `backend/internal/handler/ws.go`

- [ ] **Step 1: Update `ws.go` to handle Room Signals**

Update `backend/internal/handler/ws.go` to accept `rm *signaling.RoomManager`:

```go
type Handler struct {
	hub            *signaling.Hub
	roomMgr        *signaling.RoomManager
	metrics        *telemetry.Metrics
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

func NewHandler(hub *signaling.Hub, roomMgr *signaling.RoomManager, metrics *telemetry.Metrics, allowedOrigins []string) *Handler {
	h := &Handler{
		hub:            hub,
		roomMgr:        roomMgr,
		metrics:        metrics,
		allowedOrigins: allowedOrigins,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			return config.IsOriginAllowed(origin, h.allowedOrigins)
		},
	}

	return h
}
```

And in `ServeWS` switch statement add handling for `TypeCreateRoom`, `TypeJoinRoom`, `TypeLeaveRoom`:

```go
		switch msg.Type {
		case domain.TypeCreateRoom:
			if client == nil && msg.SenderID != "" {
				client = &wsClient{id: msg.SenderID, conn: conn}
				h.hub.Register(client)
			}
			if client != nil && h.roomMgr != nil {
				room, err := h.roomMgr.CreateRoom(msg.RoomID, client.ID())
				if err != nil {
					_ = client.Send(domain.SignalMessage{Type: domain.TypeError, Payload: err.Error()})
				} else {
					_, _ = h.roomMgr.JoinRoom(room.ID, client)
					_ = client.Send(domain.SignalMessage{Type: domain.TypeRoomCreated, RoomID: room.ID, SenderID: client.ID()})
				}
			}

		case domain.TypeJoinRoom:
			if client == nil && msg.SenderID != "" {
				client = &wsClient{id: msg.SenderID, conn: conn}
				h.hub.Register(client)
			}
			if client != nil && h.roomMgr != nil {
				_, err := h.roomMgr.JoinRoom(msg.RoomID, client)
				if err != nil {
					_ = client.Send(domain.SignalMessage{Type: domain.TypeError, Payload: err.Error()})
				}
			}

		case domain.TypeLeaveRoom:
			if client != nil && h.roomMgr != nil && msg.RoomID != "" {
				h.roomMgr.LeaveRoom(msg.RoomID, client.ID())
			}

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
			if msg.RoomID != "" && h.roomMgr != nil {
				if err := h.roomMgr.RelayRoomSignal(msg.RoomID, msg); err != nil {
					slog.Debug("Failed to relay room signal", "room_id", msg.RoomID, "error", err)
				}
			} else {
				if err := h.hub.Relay(msg); err != nil {
					if errors.Is(err, signaling.ErrPeerNotFound) {
						slog.Debug("Failed to relay message: target not found", "targetId", msg.TargetID)
					} else {
						slog.Error("Error relaying signal message", "error", err, "targetId", msg.TargetID)
					}
				}
			}

		default:
			slog.Warn("Unknown message type", "type", msg.Type)
		}
```

- [ ] **Step 2: Update `main.go`**

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

	roomMgr := signaling.NewRoomManager(metrics)

	wsHandler := handler.NewHandler(hub, roomMgr, metrics, cfg.AllowedOrigins)

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
	r.Get("/api/v1/ice-servers", handler.HandleICEServers)
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

	roomMgr.Stop()
	hub.Stop()

	slog.Info("Server shutdown completed cleanly.")
}
```

- [ ] **Step 3: Update `server_test.go`**

Update `backend/cmd/server/server_test.go` to test `/api/v1/ice-servers`.

```go
func TestICEServersEndpoint(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "stun:") {
		t.Fatalf("expected ice servers response to contain STUN URLs")
	}
}
```

- [ ] **Step 4: Build binary and run all tests with race detector**

Run: `cd /home/sthomas/projects/ezyshare/backend && go build -o /tmp/ezyshare-server ./cmd/server && go test ./... -v -race -cover`
Expected: PASS across all packages with 0 data races.

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat(server): integrate zero-knowledge RoomManager, room signaling, and /api/v1/ice-servers endpoint"
```
