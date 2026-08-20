package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders("prod")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing X-Frame-Options")
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("missing HSTS header in prod")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("missing CSP header")
	}
}
