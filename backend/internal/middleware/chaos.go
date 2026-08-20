package middleware

import (
	"crypto/subtle"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

func IsAuthorizedChaosRequest(r *http.Request) bool {
	prodSecret := os.Getenv("CHAOS_SECRET")
	if prodSecret == "" {
		return false
	}

	clientSecret := r.Header.Get("X-Chaos-Secret")
	if clientSecret == "" {
		clientSecret = r.URL.Query().Get("chaos_secret")
	}

	if clientSecret == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(clientSecret), []byte(prodSecret)) == 1
}

func GetChaosLatency(r *http.Request) time.Duration {
	if !IsAuthorizedChaosRequest(r) {
		return 0
	}

	latencyMsStr := r.Header.Get("X-Chaos-Latency-Ms")
	if latencyMsStr == "" {
		latencyMsStr = r.URL.Query().Get("chaos_latency_ms")
	}

	latencyMs, _ := strconv.Atoi(latencyMsStr)
	if latencyMs <= 0 {
		latencyMs = 300
	}

	return time.Duration(rand.Intn(latencyMs)) * time.Millisecond
}

func ShouldDropWebSocketFrame(r *http.Request) bool {
	if !IsAuthorizedChaosRequest(r) {
		return false
	}

	errorRateStr := r.Header.Get("X-Chaos-Error-Rate")
	if errorRateStr == "" {
		errorRateStr = r.URL.Query().Get("chaos_error_rate")
	}

	errorRate, _ := strconv.ParseFloat(errorRateStr, 64)
	if errorRate <= 0 {
		errorRate = 0.1
	}

	return rand.Float64() < errorRate
}

func ChaosMiddleware(metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsAuthorizedChaosRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Inject Artificial Jitter / Latency
			delay := GetChaosLatency(r)
			if delay > 0 {
				time.Sleep(delay)
			}

			// 2. Inject Artificial HTTP Fault
			errorRateStr := r.Header.Get("X-Chaos-Error-Rate")
			if errorRateStr == "" {
				errorRateStr = r.URL.Query().Get("chaos_error_rate")
			}
			errorRate, _ := strconv.ParseFloat(errorRateStr, 64)
			if errorRate <= 0 {
				errorRate = 0.1
			}

			if rand.Float64() < errorRate {
				slog.Warn("🔥 Authorized Production Chaos Injected",
					"client_ip", telemetry.GetClientIP(r),
					"path", r.URL.Path,
				)
				if metrics != nil && metrics.RateLimitExceeded != nil {
					metrics.RateLimitExceeded.WithLabelValues("chaos_injection").Inc()
				}
				http.Error(w, "500 Internal Server Error (Authorized Production Chaos Injected)", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
