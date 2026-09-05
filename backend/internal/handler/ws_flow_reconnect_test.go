package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestFullFlowRepeatedCycles(t *testing.T) {
	// Setup server
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()
	defer hub.Stop()

	roomMgr := signaling.NewRoomManager(metrics)
	wsHandler := handler.NewHandler(hub, roomMgr, metrics, []string{"*"})
	ts := httptest.NewServer(http.HandlerFunc(wsHandler.ServeWS))
	defer ts.Close()

	dial := func() (*websocket.Conn, error) {
		wsURL := "ws" + ts.URL[4:]
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		return conn, err
	}

	for i := 0; i < 3; i++ {
		// Client A creates room
		a, err := dial()
		if err != nil {
			t.Fatalf("cycle %d: dial A failed: %v", i, err)
		}
		roomID := fmt.Sprintf("room%d", i)
		create := domain.SignalMessage{Type: domain.TypeCreateRoom, SenderID: "clientA", RoomID: roomID}
		if err := a.WriteJSON(create); err != nil {
			t.Fatalf("cycle %d: create failed: %v", i, err)
		}
		// A receives the room_created ack
		var recvCreated domain.SignalMessage
		a.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err := a.ReadJSON(&recvCreated); err != nil {
			t.Fatalf("cycle %d: A read room_created failed: %v", i, err)
		}
		if recvCreated.Type != domain.TypeRoomCreated {
			t.Fatalf("cycle %d: A received wrong ack: %+v", i, recvCreated)
		}
		// Client B joins
		b, err := dial()
		if err != nil {
			t.Fatalf("cycle %d: dial B failed: %v", i, err)
		}
		join := domain.SignalMessage{Type: domain.TypeJoinRoom, SenderID: "clientB", RoomID: roomID}
		if err := b.WriteJSON(join); err != nil {
			t.Fatalf("cycle %d: join failed: %v", i, err)
		}
		// A receives the peer_joined notification for B
		var recvJoined domain.SignalMessage
		a.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err := a.ReadJSON(&recvJoined); err != nil {
			t.Fatalf("cycle %d: A read peer_joined failed: %v", i, err)
		}
		if recvJoined.Type != domain.TypePeerJoined || recvJoined.SenderID != "clientB" {
			t.Fatalf("cycle %d: A received wrong peer_joined: %+v", i, recvJoined)
		}

		// A sends offer to B
		offer := domain.SignalMessage{Type: domain.TypeOffer, SenderID: "clientA", TargetID: "clientB", Payload: "offer-data", RoomID: roomID}
		if err := a.WriteJSON(offer); err != nil {
			t.Fatalf("cycle %d: offer send failed: %v", i, err)
		}
		// B receives offer
		var recvOffer domain.SignalMessage
		b.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err := b.ReadJSON(&recvOffer); err != nil {
			t.Fatalf("cycle %d: B read offer failed: %v", i, err)
		}
		if recvOffer.Type != domain.TypeOffer || recvOffer.Payload != "offer-data" {
			t.Fatalf("cycle %d: B received wrong offer: %+v", i, recvOffer)
		}
		// B sends answer
		answer := domain.SignalMessage{Type: domain.TypeAnswer, SenderID: "clientB", TargetID: "clientA", Payload: "answer-data", RoomID: roomID}
		if err := b.WriteJSON(answer); err != nil {
			t.Fatalf("cycle %d: answer send failed: %v", i, err)
		}
		// A receives answer
		var recvAnswer domain.SignalMessage
		a.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err := a.ReadJSON(&recvAnswer); err != nil {
			t.Fatalf("cycle %d: A read answer failed: %v", i, err)
		}
		if recvAnswer.Type != domain.TypeAnswer || recvAnswer.Payload != "answer-data" {
			t.Fatalf("cycle %d: A received wrong answer: %+v", i, recvAnswer)
		}
		// Both leave
		leaveA := domain.SignalMessage{Type: domain.TypeLeaveRoom, SenderID: "clientA", RoomID: roomID}
		if err := a.WriteJSON(leaveA); err != nil {
			t.Fatalf("cycle %d: A leave failed: %v", i, err)
		}
		a.Close()
		leaveB := domain.SignalMessage{Type: domain.TypeLeaveRoom, SenderID: "clientB", RoomID: roomID}
		if err := b.WriteJSON(leaveB); err != nil {
			t.Fatalf("cycle %d: B leave failed: %v", i, err)
		}
		b.Close()
		// Allow cleanup
		time.Sleep(10 * time.Millisecond)
		if peers := testutil.ToFloat64(metrics.ActivePeers); peers != 0 {
			t.Fatalf("cycle %d: expected 0 active peers, got %f", i, peers)
		}
	}
}
