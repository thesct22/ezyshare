package handler_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
)

func TestRepeatedJoinLeaveCycles(t *testing.T) {
	ts, hub, metrics := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	for i := 0; i < 5; i++ {
		// Client A creates a room and then leaves
		connA, err := dialWS(ts.URL)
		if err != nil {
			t.Fatalf("cycle %d: failed to dial A: %v", i, err)
		}
		// Create room
		create := domain.SignalMessage{Type: domain.TypeCreateRoom, SenderID: "clientA", RoomID: fmt.Sprintf("room%d", i)}
		if err := connA.WriteJSON(create); err != nil {
			t.Fatalf("cycle %d: write create failed: %v", i, err)
		}
		// Immediately leave
		leave := domain.SignalMessage{Type: domain.TypeLeaveRoom, SenderID: "clientA", RoomID: create.RoomID}
		if err := connA.WriteJSON(leave); err != nil {
			t.Fatalf("cycle %d: write leave failed: %v", i, err)
		}
		connA.Close()
		// Give hub a moment to process cleanup
		time.Sleep(10 * time.Millisecond)
		if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
			t.Fatalf("cycle %d: expected 0 active peers after leave, got %f", i, val)
		}
	}
}

// TestBroadcastSurvivesPastPingWriteDeadline is a regression test for a bug
// where the server's 30s ping goroutine set a 10s write deadline on the
// connection and never reset it. Any room broadcast (e.g. peer_joined) sent
// more than 10s after the most recent ping - which is true for roughly 20
// of every 30 seconds - hit an already-expired deadline and failed instantly
// with "i/o timeout", silently swallowed by the caller. This reproduced the
// real-world symptom: a client leaving and rejoining a room would work
// immediately after joining, but consistently fail to notify the host once
// ~10-20s had passed since the last ping.
func TestBroadcastSurvivesPastPingWriteDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("spans a real 30s+ ping interval; skipped with -short")
	}

	ts, hub, _ := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	host, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial host: %v", err)
	}
	defer host.Close()

	create := domain.SignalMessage{Type: domain.TypeCreateRoom, SenderID: "host", RoomID: "stale-deadline-room"}
	if err := host.WriteJSON(create); err != nil {
		t.Fatalf("write create failed: %v", err)
	}
	var created domain.SignalMessage
	host.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := host.ReadJSON(&created); err != nil || created.Type != domain.TypeRoomCreated {
		t.Fatalf("host did not receive room_created ack: msg=%+v err=%v", created, err)
	}

	// Stay connected (no writes from the test) past one full ping cycle
	// (30s ping period + 10s write-deadline window) so the connection's
	// write deadline, if left stale by the ping goroutine, would have
	// already expired by the time the peer joins below.
	time.Sleep(41 * time.Second)

	peer, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial peer: %v", err)
	}
	defer peer.Close()

	join := domain.SignalMessage{Type: domain.TypeJoinRoom, SenderID: "peer", RoomID: "stale-deadline-room"}
	if err := peer.WriteJSON(join); err != nil {
		t.Fatalf("write join failed: %v", err)
	}

	var joined domain.SignalMessage
	host.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := host.ReadJSON(&joined); err != nil {
		t.Fatalf("host did not receive peer_joined broadcast after the ping window elapsed (this is the stale write-deadline bug): %v", err)
	}
	if joined.Type != domain.TypePeerJoined || joined.SenderID != "peer" {
		t.Fatalf("host received unexpected message: %+v", joined)
	}
}
