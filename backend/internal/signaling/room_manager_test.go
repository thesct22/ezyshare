package signaling_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

	if room.PeerCount() != 1 {
		t.Fatalf("expected 1 peer in room, got %d", room.PeerCount())
	}

	// Leave room
	rm.LeaveRoom("my-custom-room", c2.ID())
	if val := testutil.ToFloat64(metrics.ActiveRooms); val != 0 {
		t.Fatalf("expected 0 active rooms after empty cleanup, got %f", val)
	}
}
