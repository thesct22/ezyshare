package telemetry_test

import (
	"net/http/httptest"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func TestInitLogger(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid"}
	formats := []string{"json", "text"}

	for _, lvl := range levels {
		for _, fmtStr := range formats {
			logger := telemetry.InitLogger(lvl, fmtStr)
			if logger == nil {
				t.Fatalf("expected non-nil logger for level %s, format %s", lvl, fmtStr)
			}
		}
	}
}

func TestGetClientIPCases(t *testing.T) {
	t.Run("X-Forwarded-For multiple IPs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		if ip := telemetry.GetClientIP(req); ip != "203.0.113.195" {
			t.Fatalf("expected 203.0.113.195, got %s", ip)
		}
	})

	t.Run("X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "198.51.100.1")
		if ip := telemetry.GetClientIP(req); ip != "198.51.100.1" {
			t.Fatalf("expected 198.51.100.1, got %s", ip)
		}
	})

	t.Run("RemoteAddr host port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.50:54321"
		if ip := telemetry.GetClientIP(req); ip != "192.168.1.50" {
			t.Fatalf("expected 192.168.1.50, got %s", ip)
		}
	})

	t.Run("RemoteAddr fallback without port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "invalid-address-format"
		if ip := telemetry.GetClientIP(req); ip != "invalid-address-format" {
			t.Fatalf("expected fallback invalid-address-format, got %s", ip)
		}
	})
}
