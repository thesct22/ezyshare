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
