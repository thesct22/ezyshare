# Security Policy 🛡️

EzyShare takes security and user privacy extremely seriously. As a zero-knowledge peer-to-peer file-sharing application, security is foundational to our architecture.

---

## 🔒 Security Architecture

- **Zero-Knowledge P2P Transfer**: File payloads stream directly between browser peers via WebRTC DataChannels using **DTLS-SCTP transport layer encryption**. No file data touches backend storage or memory.
- **Client-side Encryption**: Password-protected rooms use **PBKDF2-HMAC-SHA256 (100,000 iterations)** and **AES-GCM-256** authenticated encryption via the Web Crypto API.
- **Keyless Infrastructure**: GitHub Actions CI/CD authenticates to Google Cloud via short-lived OIDC tokens using Workload Identity Federation (WIF).

---

## ⚠️ Reporting a Vulnerability

If you discover a security vulnerability within EzyShare, please **do not open a public GitHub issue**.

Instead, report security concerns directly to the project owner:

- **Primary Contact**: Sharath ([@thesct22](https://github.com/thesct22))
- **Response SLA**: We endeavor to acknowledge reports within 48 hours and provide a resolution timeline.

---

## 🛡️ Supported Versions

| Component             | Status       |
| :-------------------- | :----------- |
| `main` branch         | ✅ Supported |
| GitHub Pages Frontend | ✅ Supported |
| GCP Cloud Run Backend | ✅ Supported |
