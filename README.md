# Retail-Backbone-GCP 🚀

A resilient, edge-first architecture designed to synchronize thousands of 
logistics centers and retail stores with Google Cloud Platform.

## 🏗 The Architecture
This project demonstrates a high-availability "Hybrid Cloud" approach:
- **Edge Layer:** Go-based microservices running in local warehouses (simulated via Docker).
- **Resilience:** Local SQLite persistence for 100% offline capability during network outages.
- **Cloud Transition:** Automated event replaying to GCP Pub/Sub and BigQuery once connectivity is restored.
- **Scale:** Infrastructure as Code (Terraform) to manage global cloud resources.

## 🛠 Tech Stack
- **Language:** Go (Golang) for high-performance edge processing.
- **Database:** SQLite (Edge) & BigQuery (Cloud Analytics).
- **Orchestration:** Docker & Kubernetes (GKE).
- **Infrastructure:** Terraform (GCP Pub/Sub, IAM, BigQuery).

## 🌍 Why this scales
Instead of a monolithic approach, this "Backbone" uses a decentralized 
event-driven design. It is built to handle 3,000+ locations by treating 
each store as an autonomous edge-node.

## Project Structure

```text
retail-backbone-gcp/
├── cmd/
│   └── server/
│       └── main.go           # Entry point: Initializes DB, routes, and starts the server
├── internal/
│   └── database/
│       ├── db.go             # Postgres connection & Idempotent JSON Seeding logic
│       └── handlers.go       # Logic for /ingest, /checkout, and /status (Dashboard)
├── web/
│   └── index.html            # The Web-Scanner UI for iPhone & S25
├── inventory.json            # Master Data (extracted via Google Lens/Gemini)
├── Dockerfile                # Multi-stage build for a slim, secure container
├── docker-compose.yml        # Orchestrates the Go App + PostgreSQL Database
└── go.mod                    # Go module definition & dependencies (lib/pq)
```
