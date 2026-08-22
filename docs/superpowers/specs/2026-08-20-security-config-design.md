# Design Specification: Production Security & Environment Configuration

## Overview

This specification details adding environment-based configuration switching (`dev`, `staging`, `prod`), strict CORS origin enforcement (allowing `https://sharath.is-a.dev` for GitHub Pages frontend host in production), WebSocket origin verification, HTTP security headers, and payload size controls for the EzyShare Go backend.

## Architectural Components & Changes

### 1. Configuration Package (`internal/config`)

#### `config.go`

- `Config` struct:
  - `AppEnv` (`string`): `dev`, `staging`, or `prod` (env: `APP_ENV` or `ENV`, default: `dev`).
  - `Port` (`string`): HTTP listening port (env: `PORT`, default: `8080`).
  - `LogLevel` (`string`): Log level (`debug` in dev, `info` in staging/prod, env: `LOG_LEVEL`).
  - `LogFormat` (`string`): Log format (`text` in dev, `json` in staging/prod, env: `LOG_FORMAT`).
  - `AllowedOrigins` (`[]string`): Origins allowed for CORS and WebSocket upgrades (env: `ALLOWED_ORIGINS` as comma-separated string).
    - Defaults per `AppEnv`:
      - `prod`: `["https://sharath.is-a.dev"]`
      - `staging`: `["https://sharath.is-a.dev", "http://localhost:*", "http://127.0.0.1:*"]`
      - `dev`: `["http://localhost:*", "http://127.0.0.1:*", "https://sharath.is-a.dev"]`
- Function `LoadConfig() *Config`:
  - Parses environment variables and populates defaults.
  - Helper `IsOriginAllowed(origin string) bool`: Matches origin against wildcard patterns or exact strings.

### 2. WebSocket Handler Security (`internal/handler/ws.go`)

- Update `Handler` struct to accept `allowedOrigins []string` or `config *config.Config`.
- Update constructor: `NewHandler(hub *signaling.Hub, metrics *telemetry.Metrics, allowedOrigins []string) *Handler`.
- WebSocket `Upgrader.CheckOrigin`:
  - Validates request `Origin` header against `allowedOrigins` using pattern matching.
  - Rejects connection upgrade if origin is unapproved.
- Connection Payload Limit:
  - Set `conn.SetReadLimit(64 * 1024)` (64 KB) on upgraded connections to prevent memory exhaustion attacks.

### 3. HTTP Security & Server Entrypoint (`cmd/server/main.go`)

- Load configuration using `config.LoadConfig()`.
- Pass `cfg.AllowedOrigins` into Chi `cors.Handler`.
- Add HTTP Security Headers Middleware:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: strict-origin-when-cross-origin`
- Update HTTP server timeouts based on `APP_ENV`.

## Verification Plan

### Automated Tests

- Run `go test ./... -v -race`
- `internal/config`: Test environment loading (`dev`, `staging`, `prod`), fallback defaults, and `IsOriginAllowed` matching (`https://sharath.is-a.dev`, wildcards, unauthorized origins).
- `internal/handler`: Test WebSocket upgrade rejection with unauthorized `Origin` header and acceptance with `https://sharath.is-a.dev`.
- `cmd/server`: Test CORS preflight and security headers.
