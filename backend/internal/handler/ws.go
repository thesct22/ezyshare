package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
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
	upgrader websocket.Upgrader
}

func NewHandler(hub *signaling.Hub) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow all origins for now, will restrict in production!
			},
		},
	}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}
	defer conn.Close()

	var client *wsClient

	// Read loop: keeps listening for incoming WebSocket messages
	for {
		var msg domain.SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			slog.Error("Failed to read message", "error", err) // breaks if the connection is closed or an error occurs
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
				h.hub.Unregister(client)
				slog.Info("Peer left", "peer_id", client.ID())
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

		if client != nil {
			h.hub.Unregister(client)
			slog.Info("Peer disconnected", "peer_id", client.ID())
		}
	}

}
