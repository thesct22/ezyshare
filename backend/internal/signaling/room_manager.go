package signaling

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomIDTaken   = errors.New("custom room ID already in use")
	ErrRoomFull      = errors.New("room reached maximum peer capacity")
	ErrInvalidRoomID = errors.New("invalid custom room ID (must be 4-64 alphanumeric characters, hyphens, or underscores)")
)

var roomIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,64}$`)

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
	} else {
		if !roomIDRegex.MatchString(roomID) {
			return nil, ErrInvalidRoomID
		}
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
		room.Broadcast(msg, "")
		return nil
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
