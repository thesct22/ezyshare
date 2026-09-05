# Horizontal scaling without losing correctness — design notes

**Status:** Not started. Written for future consideration, not immediate implementation.

**Context:** On 2026-09-06, cross-person room joins in production were failing intermittently ("room not found" for a peer joining a room that definitely existed). Root cause: `RoomManager` (`backend/internal/signaling/room_manager.go`) keeps all room/peer state in a plain in-process Go map, but the Cloud Run service had `max-instances=5` with session affinity off (confirmed via `gcloud run services describe`; not set anywhere in Terraform, so likely set out-of-band via console/CLI at some point). Cloud Run gives no guarantee that two independent WebSocket connections land on the same instance. A room created on instance A is invisible to a peer routed to instance B. Cloud Run request logs for the service showed 4 distinct instance IDs active over a 7-day window, in non-overlapping windows — consistent with scale-to-zero-and-cold-restart wiping room state between sessions, not just concurrent fan-out.

**Immediate fix shipped:** `max_instance_count = 1` in `deployment/terraform/main.tf`, applied live via `gcloud run services update`. This makes the bug structurally impossible — there's only ever one process, so there's only one room map. Zero cost impact (min-instances stays 0; this only lowers a ceiling that was unnecessarily high for an app capped at 2 peers per room). Cloud Run does not scale down an instance with an open connection, so a host's room survives scale-to-zero as long as their own connection stays open — the only remaining edge case is the host's connection already having died before a guest joins, which the app already treats as an invalid room by design (`JoinRoom` rejects when the host isn't present).

**Why this isn't a long-term scaling answer:** with `max=1`, one Cloud Run instance (1 vCPU / 512Mi per `main.tf`) handles all traffic. Cloud Run's default per-instance concurrency (80 in-flight requests) means this comfortably covers dozens of simultaneous room-pairs for a personal-use tool, but it's a hard ceiling — there's no path to more capacity without revisiting the in-memory design.

## Proposed design: shared state in Firestore, instances stay stateless

The core idea: stop trying to solve cross-instance consistency ourselves. Offload it to a managed service that already solves distributed state correctly, and keep Cloud Run instances interchangeable. This is the standard "stateless compute + external state store" pattern, and it lets `max-instances` go back above 1 safely.

**Why Firestore specifically:** it has a genuine, permanent free tier (not a trial) — 1 GiB storage, 50,000 reads/day, 20,000 writes/day, 20,000 deletes/day. A room session (create, join, a handful of ICE candidates, offer, answer, leave) is roughly 10-30 document operations. At personal-project usage levels this stays free indefinitely; you'd need on the order of a thousand-plus sessions a day before approaching the free tier's limits. Alternatives considered:

- **Memorystore for Redis** — no free tier, ~$35+/month minimum even at the smallest size. Ruled out on cost alone.
- **Cloud SQL** — same problem, plus more operational overhead than this app needs.
- **Cloud Pub/Sub** — free tier is generous (10 GiB/month) and could carry the message-relay half of this, but adds a second service to manage (topic/subscription lifecycle per room) for no benefit over Firestore's realtime listeners, which can do both jobs.

**Room membership (`rooms/{roomId}` document):** `hostId`, `peerIds`, `createdAt`, `lastActive`. `CreateRoom`/`JoinRoom`/`LeaveRoom`/`KickPeer` become Firestore reads/writes instead of map operations. The "is the room full / does it already have this peer" check that currently races across three lock acquisitions in `room_manager.go` becomes a Firestore transaction, which handles that race correctly across processes for free.

**Message relay — keep the fast path, add a fallback:** this is the part worth preserving carefully. Today, `Hub.Relay` and `Room.Broadcast`/`SendTo` deliver messages by looking up a live `wsClient` in a local map and writing directly to its socket (`backend/internal/domain/room.go`, `backend/internal/signaling/hub.go`). That fast path should stay exactly as it is for the common case — both peers happen to be on the same instance, zero Firestore involvement, no added latency. Only when a target isn't found locally (today that surfaces as `ErrPeerNotFound`) should the code fall back to writing the message into a `rooms/{roomId}/signals` subcollection. Every instance holding a live connection for a given peer keeps a Firestore realtime listener scoped to `where targetId == <that peer's ID>` (at most 2 such listeners per room, since `MaxPeersPerRoom = 2`) — when a signal document appears, the owning instance delivers it down its own WebSocket and deletes the document. Sub-second Firestore listener latency is fine for SDP/ICE exchange; users already expect the initial handshake to take a moment.

**Cleanup:** Firestore's native TTL field can auto-expire abandoned `rooms/*` and `signals/*` documents, replacing the manual 30-second cleanup ticker in `RoomManager.startCleanupLoop`.

## What this buys back

- `max-instances` can return to 3+ (or whatever) without reintroducing the bug — correctness no longer depends on instance topology.
- min-instances can stay 0; Firestore is now the source of truth, not instance memory, so scale-to-zero between sessions is a non-issue.
- Cost stays effectively $0 for actual usage patterns.

## Honest scope

This is a real feature, not a config change: it touches `RoomManager`'s whole interface, the relay fallback path in `ws.go`/`hub.go`/`room.go`, adds a new dependency (Firestore Go client, service account permissions, a `google_firestore_database` + IAM binding in Terraform), and needs tests for the cross-instance fallback path specifically — which is awkward to exercise with a single test process and will likely need either two `RoomManager` instances pointed at a Firestore emulator, or a dedicated integration test against real Firestore. Rough sizing: a proper multi-hour build, not a quick patch. Recommend running this through `superpowers:brainstorming` and `superpowers:writing-plans` before touching code, given the design decisions still open below.

## Open questions to resolve before implementing

- Keep the local-fast-path optimization (added complexity, saves Firestore ops and latency when both peers share an instance) or always go through Firestore for simplicity? Given max-instances would only need to be small (3-5) for this app's realistic scale, the fast path's benefit may not be worth the extra code path to maintain.
- TTL duration for abandoned rooms/signals — current in-memory logic uses a 2-minute empty-room grace period and a 1-hour hard cap; does that same split still make sense with Firestore TTL (which is a single expiry timestamp per document, so the two-tier logic would need two different TTL fields or two collections)?
- Do metrics (`ActiveRooms`, `ActivePeers` in `backend/internal/telemetry/metrics.go`) still mean the same thing once state is shared? They're currently per-instance in-memory counters; with `max=1` today they happen to equal the global count, but that stops being true once multiple instances exist.
- Local testing story — the current test suite (`ws_test.go`, `room_manager_test.go`) spins up a real in-process server; a Firestore-backed version either needs the Firestore emulator wired into CI or a mockable storage interface behind `RoomManager`.

## Related, separate, and much simpler: TURN is not configured in production

Independent of the scaling bug above, `backend/internal/handler/ice.go` already has full ephemeral Coturn TURN credential generation, gated behind `TURN_SERVER_URL`/`TURN_SECRET` environment variables. Checking the live service (`gcloud run services describe ezyshare-backend --format="yaml(spec.template.spec.containers[0].env)"`), neither is set — the deployed `/api/v1/ice-servers` response only ever returns Google's public STUN servers. STUN alone cannot traverse symmetric NAT (common on mobile carrier networks and increasingly on residential ISPs behind CGNAT), so **even after the room-visibility bug above is fixed, some cross-network pairs may still fail to establish the actual WebRTC data channel** (signaling succeeds, both sides see the room, but the P2P connection itself never opens). This is a plausible independent contributor to "works on my LAN, fails across different networks/continents" reports. Fixing it doesn't require any of the above — just standing up a TURN server (a small coturn instance, or a managed/free-tier TURN provider) and setting the two env vars on the Cloud Run service. Worth doing as its own, much smaller follow-up.
