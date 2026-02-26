# Retail Edge - Backbone (GCP Edition)

A high-performance, resilient retail backend service written in **Go**, designed for edge computing environments. This system handles product lookups, transactions, and real-time inventory synchronization with integrated **SRE monitoring**.



## 🚀 Features

* **Atomic Transactions:** Secure `/checkout` and `/restock` operations using PostgreSQL.
* **Edge-First Security:** Full **TLS/HTTPS** support using **Tailscale** certificates for both the Go API and Grafana.
* **SRE Observability:** Native Prometheus integration tracking the **Four Golden Signals**.
* **Resilient Database Layer:** Automatic schema migration and idempotent data seeding from `inventory.json`.
* **Idempotency:** Uses `ON CONFLICT (ean) DO UPDATE` logic to ensure data consistency across restarts without duplicates.

---

## 🛠 Tech Stack

* **Backend:** Go (Golang)
* **Database:** PostgreSQL 15+
* **Infrastructure:** Docker & Docker Compose
* **Networking/Security:** Tailscale (Automated TLS)
* **Monitoring:** Prometheus & Grafana

---

## 📊 Monitoring (The Four Golden Signals)

This service exposes a `/metrics` endpoint for Prometheus. We track the health of the **Retail Edge** nodes using:

| Signal | Metric Name | Description |
| :--- | :--- | :--- |
| **Latency** | `retail_edge_latency_seconds` | P95 response time for transactions and lookups. |
| **Traffic** | `retail_edge_latency_seconds_count` | Throughput measured in requests per minute. |
| **Errors** | `retail_edge_errors_total` | Error rate (HTTP 4xx/5xx) per endpoint. |
| **Saturation** | `go_memstats_alloc_bytes` | Memory and CPU pressure on the edge node. |



---

## ⚙️ Setup & Installation

### 1. Prerequisites
* Docker & Docker Compose installed.
* Tailscale installed on the host machine.
* Tailscale certificates generated for your node:
    ```bash
    tailscale cert retail-server.tailb0ad6a.ts.net
    ```

### 2. Configuration
The system expects the following certificate files in the root directory (these are ignored by git for security):
* `retail-server.tailb0ad6a.ts.net.crt`
* `retail-server.tailb0ad6a.ts.net.key`

### 3. Run the System
```bash
docker-compose up -d
```
## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.