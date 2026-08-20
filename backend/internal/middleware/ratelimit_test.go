package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/thesct22/ezyshare/backend/internal/middleware"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestIPRateLimiter(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	limiter := middleware.NewIPRateLimiter(rate.Every(100*time.Millisecond), 2, metrics)
	defer limiter.Stop()

	handler := middleware.RateLimitMiddleware(limiter, "test_endpoint")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1: Allow
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	// Request 2: Allow (burst 2)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	// Request 3: Exceeded Rate Limit
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec3.Code)
	}

	if val := testutil.ToFloat64(metrics.RateLimitExceeded.WithLabelValues("test_endpoint")); val != 1 {
		t.Fatalf("expected rate limit metric to be 1, got %f", val)
	}
}
