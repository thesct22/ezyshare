package signaling_test

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/domain"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type mockClient struct {
	mu     sync.Mutex
	id     string
	sent   []domain.SignalMessage
	closed bool
}

func (m *mockClient) ID() string { return m.id }

func (m *mockClient) Send(msg domain.SignalMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockClient) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

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

func TestHubRegisterUnregisterConcurrent(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c := &mockClient{id: string(rune('A' + id))}
			hub.Register(c)
			hub.Unregister(c)
		}(i)
	}
	wg.Wait()

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected 0 active peers after concurrent register/unregister, got %f", val)
	}
}

func TestHubRelayErrPeerNotFound(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	msg := domain.SignalMessage{
		Type:     domain.TypeOffer,
		SenderID: "peer-1",
		TargetID: "non-existent-peer",
	}

	err := hub.Relay(msg)
	if err != signaling.ErrPeerNotFound {
		t.Fatalf("expected ErrPeerNotFound, got %v", err)
	}
}

func TestHubLifecycleAndStop(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)

	c1 := &mockClient{id: "peer-1"}
	c2 := &mockClient{id: "peer-2"}
	hub.Register(c1)
	hub.Register(c2)

	go hub.Start()
	time.Sleep(10 * time.Millisecond)

	hub.Stop()
	time.Sleep(10 * time.Millisecond)

	if !c1.isClosed() || !c2.isClosed() {
		t.Fatalf("expected all clients to be closed on hub stop")
	}

	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected 0 active peers after hub stop, got %f", val)
	}
}
