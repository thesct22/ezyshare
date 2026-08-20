package main_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func setupTestRouter() http.Handler {
	reg := prometheus.NewRegistry()
	metrics := telemetry.NewMetrics(reg)
	hub := signaling.NewHub(metrics)
	go hub.Start()
	wsHandler := handler.NewHandler(hub, metrics)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))
	r.Use(telemetry.HTTPMiddleware(metrics))

	r.Get("/ws", wsHandler.ServeWS)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	return r
}

func TestHealthzEndpoint(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "OK" {
		t.Fatalf("expected body OK, got %s", string(body))
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "ezyshare_active_peers") {
		t.Fatalf("expected metrics response to contain ezyshare_active_peers")
	}
}

func TestCORSPreflight(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("OPTIONS", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 CORS status, got %d", rec.Code)
	}
}
