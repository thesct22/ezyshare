package signaling

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

var ErrPeerNotFound = errors.New("target peer not registered")

// Hub manages active peer connections thread-safely.
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

// Register adds a new client to the hub.
func (h *Hub) Register(client domain.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[client.ID()] = client
	if h.metrics != nil {
		h.metrics.ActivePeers.Inc()
	}
	slog.Info("Peer registered in hub", "peer_id", client.ID())
}

// Unregister removes a client from the hub.
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

// Relay forwards the message to the intended recipient.
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

// Start launches the Hub's main event loop in a goroutine.
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

// Stop sends a signal to the quit channel to stop the Hub.
func (h *Hub) Stop() {
	close(h.quit)
}

// Helper method to disconnect every active client thread-safely
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
