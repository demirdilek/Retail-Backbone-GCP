# Retail Edge Backbone

This project demonstrates a modern, hybrid Cloud-Edge architecture for the retail industry. It focuses on local autonomy (Edge) and central data aggregation (GCP Backbone).

## Architecture Overview

The system is split into two main domains:

1.  **Cloud Backbone (GCP):**
    * **GKE Cluster:** Central control plane and data sink.
    * **Artifact Registry:** Centralized Docker image management.
    * **Terraform:** Infrastructure as Code (IaC) for reproducible environments.

2.  **Retail Edge (Store/Local):**
    * **Go SyncWorker:** Captures local scans and handles reliable synchronization.
    * **Docker Compose:** Local runtime environment (Postgres, Go-Service).
    * **Tailscale:** Secure WireGuard-based networking between Edge and Cloud.

## Prerequisites

To manage this environment, you need the following tools installed:

* **Go (1.21+):** For local development of the SyncWorker.
* **Docker & Docker Compose:** To run the local store environment.
* **Google Cloud SDK (gcloud):** To interact with GCP.
* **Terraform:** To manage the Cloud infrastructure.
* **kubectl:** To manage the GKE cluster.
* **Tailscale:** For secure cross-site networking.

## Infrastructure Setup (Cloud)

The Cloud Backbone is managed via Terraform. It sets up the central registry and the GKE cluster.

### 1. Authentication
Ensure you are authenticated with Google Cloud:
```bash
gcloud auth login
gcloud auth application-default login
```

2. Deployment
Navigate to the terraform directory and initialize the environment:

```bash
cd terraform-backbone
terraform init
terraform apply
```
3. Connect to Cluster
Once the apply is finished, configure kubectl to talk to your new cluster:

```bash
gcloud container clusters get-credentials retail-edge-cluster \
    --zone europe-west3-b \
    --project retail-backbone-gcp
```
### Troubleshooting

#### Permission Error: KUBECONFIG
If you encounter an error like `Unable to create private file [/etc/rancher/k3s/k3s.yaml]`:
Your environment is pointing to a restricted system path (common if K3s was previously installed). 

**Fix:**
```bash
unset KUBECONFIG
```
 Re-run the get-credentials command

## License

Distributed under the MIT License. See `LICENSE` for more information.