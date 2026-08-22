# EzyShare 🚀

[![Go Version](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat-square&logo=typescript)](https://www.typescriptlang.org)
[![GCP Cloud Run](https://img.shields.io/badge/GCP-Cloud_Run-4285F4?style=flat-square&logo=googlecloud)](https://cloud.google.com/run)
[![GitHub Pages](https://img.shields.io/badge/GitHub-Pages-222222?style=flat-square&logo=github)](https://pages.github.com)
[![Pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?style=flat-square&logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

**EzyShare** is a high-performance, **zero-knowledge, end-to-end encrypted (E2EE)** peer-to-peer file-sharing application. It enables instant, direct file transfers between web browsers without storing a single byte of file data on central servers.

---

## 🌟 Key Features

- 🔒 **Zero-Knowledge Encryption**: Optional password protection using **PBKDF2** (100,000 iterations) and **AES-GCM-256** authenticated encryption via the Web Crypto API.
- ⚡ **Direct WebRTC P2P DataChannels**: High-throughput file streaming using 64 KB chunking with dynamic flow control and transfer progress indicators.
- 🛡️ **Hardened Go Signaling Backend**: High-concurrency WebSocket signaling server with strict room capacity limits (2 peers max/room), IP rate limiting, security headers, and Prometheus metrics instrumentation.
- 📱 **Responsive Material UI + Tailwind CSS**: Clean interface featuring one-click room link copying, dynamic QR code generation for mobile devices, and active transfer metrics.
- ☁️ **Infrastructure as Code (Terraform & GCP)**: Automated infrastructure provisioning on **Google Cloud Run** with keyless **Workload Identity Federation (WIF)** and 24h Artifact Registry cleanup policies.
- 🧪 **Comprehensive CI/CD & Testing**: Vitest frontend test suite, Go backend unit/integration tests, path-filtered GitHub Actions CI pipelines, and local `pre-commit` hooks.

---

## 🏗️ System Architecture

```mermaid
sequenceDiagram
    autonumber
    actor Sender as Sender (Browser)
    participant Signal as Go Signaling Backend (GCP Cloud Run)
    actor Receiver as Receiver (Browser)

    Sender->>Signal: 1. Create Room via WebSocket (/ws)
    Signal-->>Sender: 2. Room Created (UUID / Custom ID)
    Sender->>Receiver: 3. Share Room Link / QR Code
    Receiver->>Signal: 4. Join Room via WebSocket (/ws)
    Signal-->>Sender: 5. Peer Joined Notification
    Sender<->Receiver: 6. Exchange WebRTC SDP & ICE Candidates via Signaling
    Note over Sender,Receiver: Direct WebRTC P2P DataChannel Established
    Sender->>Sender: 7. Encrypt Chunks (AES-GCM-256)
    Sender->>Receiver: 8. Stream Encrypted Chunks directly (P2P)
    Receiver->>Receiver: 9. Decrypt & Reassemble File Blob
```

---

## 🚀 Quick Start (Local Development)

### Prerequisites

- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [Go 1.27+](https://golang.org/dl/) (for local backend development)
- [Node.js 22+](https://nodejs.org/) (for local frontend development)

### Running with Docker Compose (Recommended)

Start both the Go backend signaling server and Vite React frontend concurrently:

```bash
# Build and start all services
docker compose up --build
```

- **Frontend Application**: `http://localhost:5173`
- **Backend Signaling Server**: `http://localhost:8080`
- **Backend Prometheus Metrics**: `http://localhost:8080/metrics`

---

## 🧪 Testing & Code Quality

### Running Tests Locally

#### Backend Tests (Go)

```bash
cd backend
go test -v ./...
```

#### Frontend Tests (Vitest + React Testing Library)

```bash
cd frontend
npm test
npm run lint
npm run typecheck
```

### Pre-commit Hooks

Ensure code formatting, linters, and unit test suites pass automatically before every git commit:

```bash
# 1. Install pre-commit (Ubuntu/Debian)
sudo apt install -y pre-commit

# 2. Enable git hook in repo
pre-commit install

# 3. Run manually against all files
pre-commit run --all-files
```

---

## 🔄 CI/CD Workflows

The repository uses pinned GitHub Actions workflows in `.github/workflows/`:

| Workflow File             | Trigger                            | Description                                                                                                    |
| :------------------------ | :--------------------------------- | :------------------------------------------------------------------------------------------------------------- |
| **`ci.yml`**              | PR / Push to `main`                | Path-filtered execution of frontend Vitest/oxlint, backend `go test`/`go vet`, and Prettier formatting checks. |
| **`deploy-frontend.yml`** | Push to `main` (`frontend/**`)     | Builds React SPA, injects backend secret CNAME, and deploys to `gh-pages` branch.                              |
| **`deploy-backend.yml`**  | Push to `main` (`backend/**`)      | Keylessly authenticates via WIF, builds Go container, pushes to Artifact Registry, and updates Cloud Run.      |
| **`terraform-infra.yml`** | `workflow_dispatch` (Manual admin) | Executes `terraform plan` or `terraform apply` keylessly against remote GCS state.                             |

---

## 📖 Deployment Documentation

For complete GCP foundation setup, Workload Identity Federation configuration, and GitHub Secrets lookup guides, refer to the detailed deployment documentation:

👉 **[Deployment & Infrastructure Guide (`docs/deployment.md`)](docs/deployment.md)**

---

## 🔒 Security Model

1. **Zero-Knowledge Guarantee**: File content and encryption keys never leave the client's browser unencrypted.
2. **Ephemeral Signaling**: Signaling WebSocket connections only broker WebRTC SDP offers/answers and ICE candidates. No file payloads ever pass through the signaling server.
3. **Keyless Cloud Infrastructure**: GitHub Actions authenticates to Google Cloud via short-lived OIDC tokens using Workload Identity Federation (no long-lived JSON service account keys).

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for details.
