package signaling_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type mockClient struct {
	id   string
	sent []domain.SignalMessage
}

func (m *mockClient) ID() string { return m.id }
func (m *mockClient) Send(msg domain.SignalMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}
func (m *mockClient) Close() error { return nil }

func TestHubMetricsInstrumentation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	c1 := &mockClient{id: "peer-1"}
	c2 := &mockClient{id: "peer-2"}

	hub.Register(c1)
	hub.Register(c2)

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 2 {
		t.Fatalf("expected 2 active peers, got %f", val)
	}

	msg := domain.SignalMessage{
		Type:     domain.TypeOffer,
		SenderID: "peer-1",
		TargetID: "peer-2",
	}

	if err := hub.Relay(msg); err != nil {
		t.Fatalf("unexpected relay error: %v", err)
	}

	if count := testutil.ToFloat64(metrics.MessagesRelayed.WithLabelValues("offer")); count != 1 {
		t.Fatalf("expected 1 offer message relayed, got %f", count)
	}

	hub.Unregister(c1)
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 1 {
		t.Fatalf("expected 1 active peer after unregister, got %f", val)
	}
}
