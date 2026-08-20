package telemetry_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestMetricsRegistrationAndMiddleware(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)

	metrics.ActivePeers.Inc()
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 1 {
		t.Fatalf("expected ActivePeers 1, got %f", val)
	}
	metrics.ActivePeers.Dec()
	if val := testutil.ToFloat64(metrics.ActivePeers); val != 0 {
		t.Fatalf("expected ActivePeers 0, got %f", val)
	}

	metrics.MessagesRelayed.WithLabelValues("offer").Inc()
	metrics.MessagesRelayed.WithLabelValues("answer").Inc()
	if count := testutil.ToFloat64(metrics.MessagesRelayed.WithLabelValues("offer")); count != 1 {
		t.Fatalf("expected MessagesRelayed offer 1, got %f", count)
	}

	metrics.WebSocketConnections.WithLabelValues("connected").Inc()
	metrics.WebSocketConnections.WithLabelValues("disconnected").Inc()
	if count := testutil.ToFloat64(metrics.WebSocketConnections.WithLabelValues("connected")); count != 1 {
		t.Fatalf("expected WebSocketConnections connected 1, got %f", count)
	}

	handler := telemetry.HTTPMiddleware(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if count := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "200")); count != 1 {
		t.Fatalf("expected HTTPRequestsTotal 1, got %f", count)
	}
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	if ip := telemetry.GetClientIP(req); ip != "203.0.113.195" {
		t.Fatalf("expected X-Forwarded-For IP 203.0.113.195, got %s", ip)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Real-IP", "198.51.100.1")
	if ip := telemetry.GetClientIP(req2); ip != "198.51.100.1" {
		t.Fatalf("expected X-Real-IP 198.51.100.1, got %s", ip)
	}
}
