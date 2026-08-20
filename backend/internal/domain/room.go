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
