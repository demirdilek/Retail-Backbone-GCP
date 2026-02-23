#Retail-Backbone-GCP 🚀
A resilient, microservice-based architecture designed to synchronize retail operations with Google Cloud Platform.

#🏗 The Architecture
This project demonstrates a high-availability Edge-Computing approach:

Microservices: Decoupled Go services for Web Hosting (UI delivery) and Barcode/Product Scanning.

Observability: Structured JSON logging for real-time latency tracking (SRE Standard).

Secure Networking: Zero-trust connectivity via Tailscale TLS for encrypted edge-to-cloud communication.

Persistence: PostgreSQL at the edge for robust, relational data handling.

#🛠 Tech Stack
Language: Go (Golang) – optimized for low-latency microservices.

Database: PostgreSQL – featuring idempotent seeding logic.

Security: Tailscale – automated TLS certificate management.

CI/CD: GitHub Actions – automated build verification for all services.

#📊 SRE & Observability
By using log/slog, we capture Golden Signals across our services. This allows us to pinpoint if latency is occurring in the Web Host or the Scanner service.

Standardized Log Format:

```JSON
{
  "time": "2026-02-20T13:05:01Z",
  "level": "INFO",
  "msg": "http_request",
  "service": "scanner-service",
  "method": "POST",
  "path": "/sell",
  "lat_ms": 14,
  "ua": "iPhone OS 18_7..."
}
```

#🚀 SRE Workflow
We use a unified Makefile to manage the microservice lifecycle:

make run: Boots the environment with validated TLS certificates.

make ship: Executes the SRE pipeline—Compiles, Commits, and Pushes to GitHub.

📂 Project Structure
```text
retail-backbone-gcp/
├── cmd/server/main.go   # Microservice entry: Decoupled Web & Scanner logic
├── internal/database/   # PostgreSQL connection & Seeding
├── web/index.html       # Scanner UI (Optimized for iPhone/S25)
├── Makefile             # SRE automation tool
├── .github/workflows/   # CI/CD Build pipeline
└── .gitignore           # Security: Certificates are never leaked
```