package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/middleware"
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

	// Heartbeat Ping/Pong setup to prune dead TCP connections
	pingPeriod := 30 * time.Second
	pongWait := 60 * time.Second
	writeWait := 10 * time.Second

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

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

		// Authorized Production Chaos Hook for WebSocket signaling frames
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
	}
}
