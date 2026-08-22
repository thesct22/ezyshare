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

	// Invalid custom room ID (too short / special chars) fails
	_, errInvalid := rm.CreateRoom("ab", "peer-1")
	if errInvalid != signaling.ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID, got %v", errInvalid)
	}

	_, errInjection := rm.CreateRoom("<script>alert(1)</script>", "peer-1")
	if errInjection != signaling.ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID for XSS payload, got %v", errInjection)
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
	rm.ForceCleanEmptyRooms()
	if val := testutil.ToFloat64(metrics.ActiveRooms); val != 0 {
		t.Fatalf("expected 0 active rooms after empty cleanup, got %f", val)
	}
}

func TestKickPeerAndIdempotentJoin(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	rm := signaling.NewRoomManager(metrics)
	defer rm.Stop()

	host := &mockClient{id: "host-1"}
	guest := &mockClient{id: "guest-1"}

	room, err := rm.CreateRoom("kick-test-room", host.ID())
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	_, _ = rm.JoinRoom("kick-test-room", host)

	// Join guest
	_, errJoin := rm.JoinRoom("kick-test-room", guest)
	if errJoin != nil {
		t.Fatalf("failed to join guest: %v", errJoin)
	}

	// Idempotent re-join by guest does not fail with ErrRoomFull
	_, errRejoin := rm.JoinRoom("kick-test-room", guest)
	if errRejoin != nil {
		t.Fatalf("expected idempotent re-join to succeed, got %v", errRejoin)
	}

	// Non-host trying to kick fails with ErrNotHost
	errNotHost := rm.KickPeer("kick-test-room", guest.ID(), host.ID())
	if errNotHost != signaling.ErrNotHost {
		t.Fatalf("expected ErrNotHost, got %v", errNotHost)
	}

	// Host kicks guest
	errKick := rm.KickPeer("kick-test-room", host.ID(), guest.ID())
	if errKick != nil {
		t.Fatalf("failed to kick guest: %v", errKick)
	}

	// Host remains in room (PeerCount == 1)
	if room.PeerCount() != 1 {
		t.Fatalf("expected host to remain in room (PeerCount == 1), got %d", room.PeerCount())
	}
}
