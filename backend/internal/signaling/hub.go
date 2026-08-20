package signaling

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/thesct22/ezyshare/backend/internal/domain"
)

var ErrPeerNotFound = errors.New("target peer not registered")

// Hub manages active peer connections thread-safely.
type Hub struct {
	mu       sync.RWMutex //Mutex prevents race conditions when multiple goroutines access the peers map.
	peers    map[string]domain.Client
	messages chan domain.SignalMessage
	quit     chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		peers:    make(map[string]domain.Client),
		messages: make(chan domain.SignalMessage),
		quit:     make(chan struct{}), //quit is a channel used to signal the hub to stop. Any signal sent to this channel will cause the hub to close all connections and exit its main loop.

	}
}

// Register adds a new client to the hub.
func (h *Hub) Register(client domain.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[client.ID()] = client
	slog.Info("Peer registered in hub", "peer_id", client.ID())
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client domain.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.peers[client.ID()]; exists {
		delete(h.peers, client.ID())
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

	return target.Send(msg)
}

// Start launches the Hub's main event loop in a goroutine.
// It listens for incoming messages and handles termination signals.
func (h *Hub) Start() {
	slog.Info("Hub event loop started")
	for {
		select {
		case msg := <-h.messages:
			// Process and relay messages from the channel
			if err := h.Relay(msg); err != nil {
				slog.Error("Failed to relay message", "error", err)
			}
		case <-h.quit:
			// Shutdown signal received! Close all connections and exit loop.
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
		delete(h.peers, id) //Remove the peer from the map after closing the connection
	}
}
