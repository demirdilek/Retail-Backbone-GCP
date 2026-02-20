Retail-Backbone-GCP 🚀
A resilient, edge-first architecture designed to synchronize thousands of logistics centers and retail stores with Google Cloud Platform.

🏗 The Architecture
This project demonstrates a high-availability Hybrid Cloud approach:

Edge Layer: High-performance Go microservices.

Observability: Structured JSON logging (SRE Standard) for real-time latency tracking across devices (iPhone/Android).

Secure Networking: Zero-trust connectivity via Tailscale, providing encrypted TLS tunnels without public port forwarding.

Resilience: Local persistence for 100% offline capability during network outages.

🛠 Tech Stack
Language: Go (Golang) - utilizing log/slog for structured observability.

Security: Tailscale TLS for automatic certificate management.

Infrastructure: Docker & GitHub Actions (CI/CD) for automated build verification.

Database: PostgreSQL (simulating store-local persistence).

📊 SRE & Observability
We don't just "log text"; we capture Golden Signals. The server emits structured JSON logs to stdout, ready to be ingested by ELK or Grafana Loki.

Example Log Entry:

JSON
{
  "time": "2026-02-20T13:04:47Z",
  "level": "INFO",
  "msg": "http_request",
  "method": "GET",
  "path": "/product",
  "status": 200,
  "lat_ms": 1,
  "ua": "iPhone; CPU iPhone OS 18_7..."
}
This allows us to monitor p99 latency differences between mobile hardware at the edge.

🚀 Quick Start (SRE Workflow)
1. Prerequisites
Ensure you have the Tailscale certificates in the root directory:

Bash
sudo tailscale cert your-node.ts.net
2. Run Locally
Use the provided Makefile to handle environment variables and permissions:

Bash
make run
3. Deploy/Ship
Automated build check and repository sync:

Bash
make ship
📂 Project Structure
Plaintext
retail-backbone-gcp/
├── cmd/server/main.go   # Entry point: Middleware-driven 'Golden Signals' logging
├── internal/database/   # Idempotent schema & PostgreSQL logic
├── web/index.html       # Web-Scanner UI optimized for mobile edge devices
├── Makefile             # SRE Automation for builds and deployment
├── .github/workflows/   # CI/CD: Automated Go build & test on every push
└── certs/               # (Gitignored) Tailscale TLS certificates
