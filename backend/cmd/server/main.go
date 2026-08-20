package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/thesct22/ezyshare/backend/internal/config"
	"github.com/thesct22/ezyshare/backend/internal/handler"
	customMiddleware "github.com/thesct22/ezyshare/backend/internal/middleware"
	"github.com/thesct22/ezyshare/backend/internal/signaling"
	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func main() {
	cfg := config.LoadConfig()

	_ = telemetry.InitLogger(cfg.LogLevel, cfg.LogFormat)

	slog.Info("Configuration loaded",
		"env", cfg.AppEnv,
		"port", cfg.Port,
		"allowed_origins", cfg.AllowedOrigins,
	)

	metrics := telemetry.NewMetrics(prometheus.DefaultRegisterer)

	hub := signaling.NewHub(metrics)
	go hub.Start()

	roomMgr := signaling.NewRoomManager(metrics)

	wsHandler := handler.NewHandler(hub, roomMgr, metrics, cfg.AllowedOrigins)

	// IP Rate Limiters
	apiLimiter := customMiddleware.NewIPRateLimiter(rate.Every(time.Minute/60), 20, metrics) // 60 req/min, burst 20
	wsLimiter := customMiddleware.NewIPRateLimiter(rate.Every(time.Minute/10), 5, metrics)   // 10 WS upgrades/min, burst 5
	defer apiLimiter.Stop()
	defer wsLimiter.Stop()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.SecurityHeaders(cfg.AppEnv))
	r.Use(customMiddleware.MaxBytesMiddleware(4 * 1024)) // Cap HTTP request body at 4KB

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(telemetry.HTTPMiddleware(metrics))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start).String(),
				"client_ip", telemetry.GetClientIP(r),
				"req_id", middleware.GetReqID(r.Context()),
			)
		})
	})

	r.With(customMiddleware.RateLimitMiddleware(wsLimiter, "ws_upgrade")).Get("/ws", wsHandler.ServeWS)
	r.With(customMiddleware.RateLimitMiddleware(apiLimiter, "healthz")).Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.With(customMiddleware.RateLimitMiddleware(apiLimiter, "ice_servers")).Get("/api/v1/ice-servers", handler.HandleICEServers)
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	slog.Info("Received shutdown signal. Stopping services...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	roomMgr.Stop()
	hub.Stop()

	slog.Info("Server shutdown completed cleanly.")
}
