# Design Specification: Backend Unit & Integration Test Suite

## Overview
This specification details adding full unit and integration test coverage across all backend packages (`telemetry`, `signaling`, `handler`, and `cmd/server`). The test suite will verify logging initialization, client IP resolution, Prometheus metrics collection, signaling Hub concurrency & message relaying, WebSocket connection upgrades & P2P signaling exchanges, and HTTP server endpoints (`/healthz`, `/metrics`, CORS).

## Architectural Components & Test Cases

### 1. Telemetry Package (`internal/telemetry`)

#### `logger_test.go`
* `TestInitLogger`:
  * Test logger creation with levels `debug`, `info`, `warn`, `error` and formats `json`, `text`.
* `TestGetClientIP`:
  * `X-Forwarded-For`: multiple comma-separated IPs (e.g. `203.0.113.195, 70.41.3.18`), verify first non-empty trimmed IP is returned.
  * `X-Real-IP`: single IP address, verify header parsed.
  * `RemoteAddr`: IP with port (`192.168.1.1:54321`), verify host extracted via `net.SplitHostPort`.
  * Fallback: invalid remote addr format.

#### `metrics_test.go`
* `TestMetricsCollectors`:
  * Verify `ActivePeers` Gauge increments and decrements.
  * Verify `MessagesRelayed` CounterVec tracks messages per type label (`offer`, `answer`, `candidate`, `join`, `leave`).
  * Verify `WebSocketConnections` CounterVec tracks status labels (`connected`, `disconnected`, `failed`).
  * Verify `HTTPRequestsTotal` CounterVec and `HTTPRequestDuration` HistogramVec collect request stats via `HTTPMiddleware`.

### 2. Signaling Package (`internal/signaling`)

#### `hub_test.go`
* `TestHubRegisterUnregister`:
  * Test registering multiple mock clients concurrently using goroutines.
  * Test unregistering clients and verifying `ActivePeers` gauge.
* `TestHubRelay`:
  * Test successful message relay between registered clients.
  * Test relaying message to unregistered target returns `ErrPeerNotFound` and logs warning.
* `TestHubEventLoopAndShutdown`:
  * Start `Hub` event loop via `go hub.Start()`.
  * Send messages through internal channel or relay method.
  * Call `hub.Stop()` and verify `closeAllConnections` closes active clients and terminates loop cleanly.

### 3. Handler Package (`internal/handler`)

#### `ws_test.go`
* Setup: Create `httptest.NewServer` wrapping `Handler.ServeWS`.
* `TestWebSocketUpgradeAndJoin`:
  * Connect using `websocket.DefaultDialer`.
  * Send `TypeJoin` message with `SenderID = "client-1"`.
  * Verify client registered in Hub.
* `TestWebSocketSignalingExchange`:
  * Dial Client A (`client-1`) and Client B (`client-2`).
  * Both clients send `TypeJoin`.
  * Client A sends `TypeOffer` targeted to `client-2`.
  * Client B reads JSON frame and asserts received `TypeOffer` with `SenderID = "client-1"`.
  * Client B replies with `TypeAnswer` targeted to `client-1`.
  * Client A reads JSON frame and asserts received `TypeAnswer`.
* `TestUnauthenticatedSignalingFrame`:
  * Dial Client C without sending `TypeJoin`.
  * Send `TypeOffer`. Verify frame dropped without crashing server.
* `TestWebSocketDisconnect`:
  * Close client connection. Verify `Unregister` called and `WebSocketConnections` metric logged `disconnected`.

### 4. Server Package (`cmd/server`)

#### `server_test.go`
* `TestHealthzEndpoint`:
  * Send `GET /healthz`. Verify `200 OK` status and body `OK`.
* `TestMetricsEndpoint`:
  * Send `GET /metrics`. Verify `200 OK` status and Prometheus format output containing `ezyshare_active_peers`, `ezyshare_http_requests_total`.
* `TestCORSHeaders`:
  * Send `OPTIONS /ws` with `Origin: http://localhost:3000`. Verify CORS headers set.

## Verification Plan

### Automated Tests
* Run `cd backend && go test ./... -v -cover`
* Verify total test coverage across backend code is 80%+ and all tests pass with zero race conditions (`go test -race ./...`).
