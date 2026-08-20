package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	// Gauge metrics
	ActivePeers prometheus.Gauge
	ActiveRooms prometheus.Gauge
	// Counter metrics
	MessagesRelayed      *prometheus.CounterVec
	WebSocketConnections *prometheus.CounterVec
	HTTPRequestsTotal    *prometheus.CounterVec
	RoomsCreatedTotal    *prometheus.CounterVec
	RateLimitExceeded    *prometheus.CounterVec
	// Histogram metric
	HTTPRequestDuration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ActivePeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ezyshare_active_peers",
			Help: "Current number of active WebRTC signaling peers connected.",
		}),
		ActiveRooms: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ezyshare_active_rooms",
			Help: "Current number of active transient signaling rooms.",
		}),
		MessagesRelayed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_messages_relayed_total",
				Help: "Total number of WebRTC signaling messages relayed.",
			},
			[]string{"type"},
		),
		WebSocketConnections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_websocket_connections_total",
				Help: "Total number of WebSocket connection events.",
			},
			[]string{"status"},
		),
		RoomsCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_rooms_created_total",
				Help: "Total number of rooms created.",
			},
			[]string{"type"},
		),
		RateLimitExceeded: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_rate_limit_exceeded_total",
				Help: "Total number of rate limit exceeded events.",
			},
			[]string{"endpoint"},
		),
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ezyshare_http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ezyshare_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}

	if reg != nil {
		reg.MustRegister(
			m.ActivePeers,
			m.ActiveRooms,
			m.MessagesRelayed,
			m.WebSocketConnections,
			m.RoomsCreatedTotal,
			m.RateLimitExceeded,
			m.HTTPRequestsTotal,
			m.HTTPRequestDuration,
		)
	}

	return m
}

func HTTPMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(ww.Status())
			path := r.URL.Path

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}
