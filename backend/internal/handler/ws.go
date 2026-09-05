package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/middleware"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

// writeWait bounds every write to the connection. It must be set fresh
// immediately before each write (ping or application message) rather than
// relying on a deadline set by a previous write - net.Conn deadlines are
// absolute wall-clock times that don't reset themselves, so a deadline left
// over from an earlier write silently expires and fails all later writes
// with "i/o timeout" once more than writeWait has elapsed since it was set.
const writeWait = 10 * time.Second

type wsClient struct {
	id      string
	conn    *websocket.Conn
	roomID  string
	writeMu sync.Mutex
}

func (c *wsClient) ID() string { return c.id }

func (c *wsClient) Send(msg domain.SignalMessage) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteJSON(msg)
}

func (c *wsClient) Close() error { return c.conn.Close() }

type Handler struct {
	hub            *signaling.Hub
	roomMgr        *signaling.RoomManager
	metrics        *telemetry.Metrics
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

func NewHandler(hub *signaling.Hub, roomMgr *signaling.RoomManager, metrics *telemetry.Metrics, allowedOrigins []string) *Handler {
	h := &Handler{hub: hub, roomMgr: roomMgr, metrics: metrics, allowedOrigins: allowedOrigins}
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
	pongWait := 120 * time.Second

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	var client *wsClient

	// Ping goroutine – protect writes with client mutex
	go func() {
		for range pingTicker.C {
			if client != nil {
				client.writeMu.Lock()
				_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					client.writeMu.Unlock()
					return
				}
				client.writeMu.Unlock()
			}
		}
	}()

	if h.metrics != nil {
		h.metrics.WebSocketConnections.WithLabelValues("connected").Inc()
	}

	defer func() {
		if client != nil {
			if client.roomID != "" && h.roomMgr != nil {
				h.roomMgr.LeaveRoom(client.roomID, client.ID())
			}
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

		// Refresh read deadline on every active message
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))

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
					client.roomID = room.ID
					_ = client.Send(domain.SignalMessage{Type: domain.TypeRoomCreated, RoomID: room.ID, SenderID: client.ID()})
				}
			}

		case domain.TypeJoinRoom:
			if client == nil && msg.SenderID != "" {
				client = &wsClient{id: msg.SenderID, conn: conn}
				h.hub.Register(client)
			}
			if client != nil && h.roomMgr != nil {
				room, err := h.roomMgr.JoinRoom(msg.RoomID, client)
				if err != nil {
					_ = client.Send(domain.SignalMessage{Type: domain.TypeError, Payload: err.Error()})
				} else {
					client.roomID = msg.RoomID
					if room != nil && room.HostID == client.ID() {
						_ = client.Send(domain.SignalMessage{Type: domain.TypeRoomCreated, RoomID: room.ID, SenderID: client.ID()})
					}
				}
			}

		case domain.TypeLeaveRoom:
			if client != nil && h.roomMgr != nil && msg.RoomID != "" {
				h.roomMgr.LeaveRoom(msg.RoomID, client.ID())
				if client.roomID == msg.RoomID {
					client.roomID = ""
				}
			}

		case domain.TypeRemovePeer:
			if client != nil && h.roomMgr != nil && msg.RoomID != "" && msg.TargetID != "" {
				err := h.roomMgr.KickPeer(msg.RoomID, client.ID(), msg.TargetID)
				if err != nil {
					_ = client.Send(domain.SignalMessage{Type: domain.TypeError, Payload: err.Error()})
				}
			}

		case domain.TypeJoin:
			if msg.SenderID == "" {
				slog.Warn("Join attempt missing senderId")
				continue
			}
			client = &wsClient{id: msg.SenderID, conn: conn}
			h.hub.Register(client)

		case domain.TypeLeave:
			if client != nil {
				slog.Info("Peer requested leave", "peer_id", client.ID())
			}
			return

		case domain.TypePing:
			// Application-level keepalive; the read deadline was already
			// refreshed above. Nothing else to do.

		case domain.TypeOffer, domain.TypeAnswer, domain.TypeCandidate, domain.TypeRequestOffer:
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
