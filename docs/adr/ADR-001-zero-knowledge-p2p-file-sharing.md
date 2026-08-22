# ADR-001: Zero-Knowledge Peer-to-Peer File Sharing Architecture

## Status

**Accepted**

## Context

EzyShare aims to provide a secure, high-speed file sharing protocol across web browsers and mobile applications (iOS/Android). Users need to share files directly without uploading file contents to cloud servers or risking server-side data exposure, snooping, or privacy breaches.

A critical design requirement is that the backend infrastructure must operate under a **Zero-Knowledge / Zero-Trust model**:

1. The backend must have **zero knowledge** of what files are being shared (file names, sizes, or contents).
2. The backend must have **zero knowledge** of room passwords or passphrases.
3. File listings and downloads must occur entirely peer-to-peer (P2P).
4. Senders must be able to use custom Room UIDs or system-generated UUIDs, share room links or QR codes, and protect rooms with passwords.

## Architectural Decision

We decide to implement a **Zero-Knowledge WebRTC Signaling & P2P Protocol**:

### 1. Blind Signaling Server Role

The Go backend acts purely as a blind WebRTC signaling broker and room router:

- Maintains transient in-memory mappings between `room_id` (custom UID or generated UUID) and connected WebSocket client peers.
- Relays encrypted WebRTC signaling frames (`offer`, `answer`, `candidate`, `join_room`, `leave_room`).
- **Does NOT store or process**:
  - File metadata (file names, file types, file sizes).
  - Passwords or password hashes.
  - Transferred file binary chunks.

### 2. Client-Side Cryptographic Authentication & File Listing

- **Password Key Derivation**: Both Sender and Receiver derive an encryption key $K$ locally using **Argon2id** or **PBKDF2** with the `room_id` as salt:
  $$K = \text{Argon2id}(\text{Password}, \text{RoomID})$$
- **P2P Authentication Handshake**: Password verification occurs directly between Receiver and Sender over the WebRTC DataChannel (or encrypted signaling payloads).
- **P2P File Listing Transfer**: On successful password verification, the Sender transmits the shared file metadata list directly to the Receiver over the WebRTC DataChannel. The Receiver inspects files in their UI and clicks to initiate P2P chunk downloads.

### 3. Ephemeral TURN Relaying for Mobile Networks

- To support mobile 4G/5G networks, corporate Wi-Fi, and symmetric NAT environments where direct STUN P2P connections are blocked, the backend exposes a secure `GET /api/v1/ice-servers` endpoint.
- Returns short-lived (ephemeral) TURN server credentials (HMAC-SHA1 generated for Coturn/TURN services).

### 4. Room Lifecycle & Rate Limiting Controls

- **Transient Rooms**: Rooms are created in-memory when a Sender connects and auto-expire when all peers disconnect or after a configurable idle timeout.
- **Security & DoS Protection**: Backend enforces rate limits on room creation per IP and limits maximum concurrent peer connections per room.

## Protocol Flow Diagram

```text
[ Sender App ]                             [ Blind Backend ]                         [ Receiver App ]
      |                                           |                                         |
      |-- 1. Create Room ("my-room") ------------>|                                         |
      |   (Backend stores ONLY "my-room" ID)      |<-- 2. Join Room ("my-room") ------------|
      |                                           |                                         |
      |<-- 3. Relay "Peer Joined" signal ---------|---------------------------------------->|
      |                                           |                                         |
      |============== 4. WebRTC P2P Handshake (Encrypted with Key K) ======================>|
      |   Key K = Argon2id(Password, RoomID)      |                                         |
      |                                           |                                         |
      |==================== 5. WebRTC P2P DataChannel Established =========================>|
      |                                           |                                         |
      |-------------------- 6. P2P Password Verification (HMAC Check) --------------------->|
      |                                           | (If Auth Valid)                         |
      |-------------------- 7. P2P Send Shared File Metadata List ------------------------->|
      |                                           | (Receiver views file list in UI)        |
      |                                           |                                         |
      |<=================== 8. P2P Chunked File Stream (64KB Chunks) ======================>|
```

## Consequences

### Positive

- **Complete Data Privacy**: Backend operator, hosting provider, or compromised server cannot read file names, file contents, or room passwords.
- **Subpoena & Legal Immunity**: Server owner cannot comply with data requests for file contents because the server never possessed or logged them.
- **Zero Server Storage Overhead**: Zero database or blob storage required on backend.

### Negative / Trade-offs

- **Simultaneous Online Requirement**: Both Sender and Receiver must be online at the same time to complete transfers.
- **TURN Relay Cost**: When direct STUN fails on restrictive mobile 4G/5G networks, WebRTC traffic falls back to TURN relay bandwidth.
