package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func setupTestServer() (*httptest.Server, *signaling.Hub, *telemetry.Metrics) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()

	roomMgr := signaling.NewRoomManager(metrics)
	wsHandler := handler.NewHandler(hub, roomMgr, metrics, []string{"*"})

	ts := httptest.NewServer(http.HandlerFunc(wsHandler.ServeWS))
	return ts, hub, metrics
}

func dialWS(url string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	return conn, err
}

func TestWebSocketOriginValidation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()
	defer hub.Stop()

	roomMgr := signaling.NewRoomManager(metrics)
	wsHandler := handler.NewHandler(hub, roomMgr, metrics, []string{"https://sharath.is-a.dev"})
	ts := httptest.NewServer(http.HandlerFunc(wsHandler.ServeWS))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Dial with unauthorized origin
	headerFail := make(http.Header)
	headerFail.Set("Origin", "https://unauthorized-site.com")
	_, respFail, errFail := websocket.DefaultDialer.Dial(wsURL, headerFail)
	if errFail == nil || respFail == nil || respFail.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthorized origin")
	}

	// Dial with authorized origin
	headerSuccess := make(http.Header)
	headerSuccess.Set("Origin", "https://sharath.is-a.dev")
	connSuccess, _, errSuccess := websocket.DefaultDialer.Dial(wsURL, headerSuccess)
	if errSuccess != nil {
		t.Fatalf("expected origin https://sharath.is-a.dev to succeed: %v", errSuccess)
	}
	connSuccess.Close()
}

func TestWebSocketSignalingExchange(t *testing.T) {
	ts, hub, metrics := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	connA, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client A: %v", err)
	}
	defer connA.Close()

	connB, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial Client B: %v", err)
	}
	defer connB.Close()

	joinA := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-a"}
	if err := connA.WriteJSON(joinA); err != nil {
		t.Fatalf("Client A join write failed: %v", err)
	}

	joinB := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-b"}
	if err := connB.WriteJSON(joinB); err != nil {
		t.Fatalf("Client B join write failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 2 {
		t.Fatalf("expected 2 active peers in hub, got %f", val)
	}

	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "client-b",
		Payload:  "offer-sdp-data",
	}
	if err := connA.WriteJSON(offer); err != nil {
		t.Fatalf("Client A offer write failed: %v", err)
	}

	var receivedOffer domain.SignalMessage
	connB.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connB.ReadJSON(&receivedOffer); err != nil {
		t.Fatalf("Client B read offer failed: %v", err)
	}

	if receivedOffer.Type != domain.TypeOffer || receivedOffer.SenderID != "client-a" {
		t.Fatalf("Client B received invalid offer: %+v", receivedOffer)
	}

	answer := domain.SignalMessage{
		Type:     domain.TypeAnswer,
		TargetID: "client-a",
		Payload:  "answer-sdp-data",
	}
	if err := connB.WriteJSON(answer); err != nil {
		t.Fatalf("Client B answer write failed: %v", err)
	}

	var receivedAnswer domain.SignalMessage
	connA.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := connA.ReadJSON(&receivedAnswer); err != nil {
		t.Fatalf("Client A read answer failed: %v", err)
	}

	if receivedAnswer.Type != domain.TypeAnswer || receivedAnswer.SenderID != "client-b" {
		t.Fatalf("Client A received invalid answer: %+v", receivedAnswer)
	}
}

func TestUnauthenticatedSignalingFrame(t *testing.T) {
	ts, hub, _ := setupTestServer()
	defer ts.Close()
	defer hub.Stop()

	conn, err := dialWS(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial WS: %v", err)
	}
	defer conn.Close()

	offer := domain.SignalMessage{
		Type:     domain.TypeOffer,
		TargetID: "some-target",
	}
	if err := conn.WriteJSON(offer); err != nil {
		t.Fatalf("write offer failed: %v", err)
	}

	join := domain.SignalMessage{Type: domain.TypeJoin, SenderID: "client-unauth"}
	if err := conn.WriteJSON(join); err != nil {
		t.Fatalf("write join failed: %v", err)
	}
}
