# Design Specification: Slog Logger & Prometheus Metrics Integration

## Overview

This specification details adding structured logging (`slog`) and Prometheus metrics instrumentation to the EzyShare backend. It integrates with existing HTTP endpoints (`chi`), WebSockets (`gorilla/websocket`), and signaling hub (`Hub`).

## Architectural Components

### 1. Telemetry Package (`internal/telemetry`)

#### `logger.go`

- Function `InitLogger(levelStr string, formatStr string) *slog.Logger`
- Environment-configurable log format (`json` or `text`) and level (`debug`, `info`, `warn`, `error`).
- Sets default global logger using `slog.SetDefault(logger)`.
- Helper function `GetClientIP(r *http.Request) string` to accurately extract client IP addresses (checking `X-Forwarded-For`, `X-Real-IP`, and fallback to `RemoteAddr` host), replacing deprecated `realip` middleware.

#### `metrics.go`

- `Metrics` struct holding Prometheus collectors (covering Counter, Gauge, and Histogram types):
  - **Gauge**: `ActivePeers` (`prometheus.Gauge`): Current number of connected WebRTC peers in `Hub`. Name: `ezyshare_active_peers`.
  - **Counters**:
    - `MessagesRelayed` (`*prometheus.CounterVec`): Count of WebRTC signaling messages relayed. Name: `ezyshare_messages_relayed_total`, label: `type` (`join`, `leave`, `offer`, `answer`, `candidate`).
    - `WebSocketConnections` (`*prometheus.CounterVec`): Count of WebSocket connection attempts. Name: `ezyshare_websocket_connections_total`, label: `status` (`connected`, `disconnected`, `failed`).
    - `HTTPRequestsTotal` (`*prometheus.CounterVec`): Count of HTTP requests. Name: `ezyshare_http_requests_total`, labels: `method`, `path`, `status`.
  - **Histogram**:
    - `HTTPRequestDuration` (`*prometheus.HistogramVec`): HTTP request duration in seconds. Name: `ezyshare_http_request_duration_seconds`, labels: `method`, `path`.
- `NewMetrics(reg prometheus.Registerer) *Metrics`
- `HTTPMiddleware(metrics *Metrics) func(http.Handler) http.Handler` - Chi middleware capturing HTTP request duration, status code, and count.

### 2. Signaling Hub Instrumentation (`internal/signaling/hub.go`)

- Modify `Hub` struct to accept `metrics *telemetry.Metrics`.
- Update constructor: `NewHub(metrics *telemetry.Metrics) *Hub`.
- Update `Register`: increment `ActivePeers.Inc()`.
- Update `Unregister`: decrement `ActivePeers.Dec()`.
- Update `Relay`: increment `MessagesRelayed.WithLabelValues(string(msg.Type)).Inc()`.

### 3. WebSocket Handler Instrumentation (`internal/handler/ws.go`)

- Modify `Handler` struct to accept `metrics *telemetry.Metrics`.
- Update constructor: `NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics) *Handler`.
- Instrument `ServeWS`:
  - Upgrade failure: `WebSocketConnections.WithLabelValues("failed").Inc()`.
  - Successful connection: `WebSocketConnections.WithLabelValues("connected").Inc()`.
  - Connection close: `WebSocketConnections.WithLabelValues("disconnected").Inc()`.
- Fix client disconnection bug: Move `Unregister` cleanup out of the message loop onto connection exit / `defer`.

### 4. HTTP Server & Entrypoint (`cmd/server/main.go`)

- Initialize logger using `telemetry.InitLogger(...)`.
- Initialize metrics with `prometheus.DefaultRegisterer`.
- Mount `/metrics` endpoint using `promhttp.Handler()`.
- Mount metrics middleware (`telemetry.HTTPMiddleware`) alongside structured request logging incorporating `telemetry.GetClientIP(r)`.
- Pass `metrics` into `signaling.NewHub(metrics)` and `handler.NewHandler(hub, metrics)`.

## Verification Plan

### Automated Tests

- Run `go test ./...` in `backend` to ensure code compiles and passes tests.

### Manual / Integration Verification

- Run `go mod tidy` in `backend/` to download required dependencies (`go-chi/chi`, `prometheus/client_golang`, etc.).
- Run `go build -o server ./cmd/server` to confirm compilation.
- Launch server and verify `/metrics` endpoint returns Prometheus metric definitions (`ezyshare_active_peers`, `ezyshare_messages_relayed_total`, `ezyshare_websocket_connections_total`, `ezyshare_http_requests_total`, `ezyshare_http_request_duration_seconds`).
- Verify JSON structured logs printed to stdout on incoming requests contain `client_ip`.
