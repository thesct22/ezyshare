package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/middleware"
)

func TestChaosMiddlewareUnauthorized(t *testing.T) {
	os.Setenv("CHAOS_SECRET", "my-secret-key")
	defer os.Unsetenv("CHAOS_SECRET")

	handler := middleware.ChaosMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unauthorized request without secret header, got %d", rec.Code)
	}
}

func TestChaosMiddlewareAuthorized(t *testing.T) {
	os.Setenv("CHAOS_SECRET", "my-secret-key")
	defer os.Unsetenv("CHAOS_SECRET")

	handler := middleware.ChaosMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/ice-servers", nil)
	req.Header.Set("X-Chaos-Secret", "my-secret-key")
	req.Header.Set("X-Chaos-Error-Rate", "1.0") // 100% error rate for test
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error for authorized chaos request, got %d", rec.Code)
	}
}
