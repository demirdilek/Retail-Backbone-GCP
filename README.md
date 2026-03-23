# Retail Edge Backbone (GCP & Tailscale)

![Go Quality Check](https://github.com/demirdilek/Retail-Backbone-GCP/actions/workflows/go.yml/badge.svg)
![Deployment Status](https://github.com/demirdilek/Retail-Backbone-GCP/actions/workflows/deploy.yml/badge.svg)

This project demonstrates a professional, hybrid Cloud-Edge architecture designed for the retail industry. It focuses on high-performance data synchronization, local store autonomy, and a **Zero Trust** security model.

---

## Core Pillars

* **Cloud Backbone (GCP):** Centralized control plane and data aggregation using GKE and Artifact Registry.
* **Retail Edge (Store):** Lightweight, autonomous nodes running **K3s** for local resilience and low-latency processing.
* **Zero Trust Mesh:** Secure, identity-based communication via **Tailscale (WireGuard)**, eliminating public IP exposure.
* **SRE Focused:** Built with the **4 Golden Signals** (Latency, Traffic, Errors, Saturation) in mind using Go-native instrumentation.

## Architecture Overview

The system bridges the gap between decentralized retail locations and a centralized Google Cloud environment:

1.  **Infrastructure as Code:** Fully managed via **Terraform** for reproducible environments.
2.  **Networking:** A private overlay mesh connects all nodes. Edge nodes join the network automatically using **OAuth2** authentication.
3.  **Security:** Automated TLS (Let's Encrypt) for all internal services via Tailscale, enabling secure browser-native APIs (e.g., Camera/Scanner) in-store.

## Architecture Decisions (ADR)

### ADR 001: Switching from GKE to k3s for Edge Nodes
* **Context:** Local simulation of edge nodes using GKE resulted in ~80% CPU saturation on development hardware.
* **Decision:** Replaced GKE with **k3s** for edge environments.
* **Consequence:** Reduced resource footprint significantly while maintaining Kubernetes API compatibility, better mirroring actual retail hardware constraints.

### ADR 002: Multi-Stage Distroless Builds
* **Context:** Retail store connectivity can be unstable or bandwidth-limited.
* **Decision:** Implemented multi-stage Docker builds using `gcr.io/distroless/static`.
* **Consequence:** Reduced image size from ~800MB to ~20MB, improving deployment speed and security (reduced attack surface).

## Observability & SRE
This project is instrumented to monitor the **4 Golden Signals**:
* **Latency:** Tracking sync duration between Edge and GCP.
* **Traffic:** Measuring the number of processed retail transactions.
* **Errors:** Monitoring 5xx rates and failed sync attempts.
* **Saturation:** Observing CPU/Memory pressure on k3s nodes.

## Quick Start (Automated via Makefile)

The project uses a `Makefile` to abstract complex workflows. Ensure you have the GCP SDK and Tailscale installed.

## 1. Initialize Infrastructure
```bash
make infra-init
make infra-up
```
**Note on Security:** This repository uses **Gitleaks** in the CI pipeline. All previously exposed test keys have been revoked, rotated, and purged from the git history using `git filter-repo`.