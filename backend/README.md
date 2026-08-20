# EzyShare Backend

This is a Golang server for the EzyShare file sharing application.
The server works as an identity broker, presence manager and signaling relay for the P2P WebRTC connections between clients.

## Key Responsibilities
1. **Client Identity & Session Management**: Assigning or validating unique peer IDs (UUIDs) when clients connect.
2. **Signaling Exchange**: Transporting WebRTC handshakes (OFFER, ANSWER, and ICE candidates) between two peers.
3. **Session Lifecycle & Cleanup**: Pruning inactive connections, handling abrupt disconnections, and expiring orphaned rooms after a timeout.
4. **Rate Limiting & Defensive Controls**: Blocking malicious bots, preventing connection exhaustion attacks, and enforcing CORS policies.
5. **Observability & Diagnostics**: Exposing operational metrics (active connections, signaling latency, drops) and health checks for orchestrators.

## Structure

```text
backend/
├── cmd/
│   └── server/
│       └── main.go           # Application entrypoint & dependency injection
├── internal/
│   ├── domain/               # Pure business models & interface definitions
│   │   ├── peer.go
│   │   └── signal.go
│   ├── handler/              # Transport layer (HTTP/WebSocket endpoints)
│   │   ├── middleware.go     # CORS, Rate Limiting, Chaos injection
│   │   └── ws.go            # WebSocket connection upgrade & handling
│   ├── signaling/            # Core business logic (Hub, Session state, Routing)
│   │   └── hub.go
│   └── config/               # Environment variables & configuration parsing
│       └── config.go
├── Dockerfile                # Production multi-stage scratch build
├── go.mod
├── go.sum
└── Makefile                  # Developer workflow automation
```

## Development

To build and run the server locally:

```bash
make clean

make build
./bin/ezyshare-server
```

## Production

To build for production:

```bash
make clean


make build-prod
docker run -p 8080:8080 ezyshare-server:latest
```

## Deployment

We are planning to deploy this to Koyeb for utilizing their free tier. 

The backend is containerized and can be deployed to any container orchestrator.

## Observability

The backend exposes metrics at `/metrics` endpoint for Prometheus to scrape.
