# Contributing to EzyShare 🚀

Thank you for your interest in contributing to EzyShare! We welcome community contributions, bug reports, and feature proposals.

---

## 📜 Code of Conduct & Principles

1. **Zero-Knowledge First**: EzyShare prioritizes user privacy. No file payloads or unencrypted user data should ever pass through or be logged by the signaling server.
2. **Security & Quality**: All contributions must pass linters, typechecks, unit tests, and pre-commit checks.

---

## 🛠️ Local Development Setup

### 1. Prerequisites

- **Node.js 22+**
- **Go 1.27+**
- **Docker & Docker Compose**
- **Pre-commit CLI** (`sudo apt install pre-commit`)

### 2. Environment Setup

```bash
# Clone the repository
git clone https://github.com/thesct22/ezyshare.git
cd ezyshare

# Enable local pre-commit hooks
pre-commit install

# Start local dev environment
docker compose up --build
```

---

## 🧪 Testing & Verification

Before submitting a Pull Request, ensure all tests pass locally:

### Frontend

```bash
cd frontend
npm ci
npm run lint       # Oxlint check
npm run typecheck  # TypeScript check
npm test           # Vitest unit tests
npm run build      # Production bundle check
```

### Backend

```bash
cd backend
gofmt -s -l .      # Format check
go vet ./...       # Vet check
go test -v ./...   # Go unit tests
```

### Repository-wide Pre-commit

```bash
pre-commit run --all-files
```

---

## 🔀 Pull Request Process

1. **Fork & Branch**: Create a feature branch from `main` (`feature/your-feature-name` or `fix/your-fix-name`).
2. **Commit Guidelines**: Use conventional commit format (e.g., `feat(frontend): ...`, `fix(backend): ...`, `docs: ...`).
3. **PR Submission**: Open a Pull Request targeting `main`.
4. **CI Pipeline**: Automated GitHub Actions CI will run your tests.
5. **Code Review**: Pull requests require explicit approval from the repository owner (`@thesct22`) before merging.

---

Thank you for building secure, zero-knowledge file sharing!
