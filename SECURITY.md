i# Security Architecture & Zero Trust Networking

This project implements a **Zero Trust security model**. Trust is not granted based on network location (e.g., "inside the GCP VPC") but is instead defined by service identity and strong encryption.

---

## 1. Networking via Tailscale Mesh
Rather than relying on legacy VPN gateways or risky port forwarding, communication between the **Retail Edge Nodes** and the **GCP Backbone** is built on a Tailscale mesh network (powered by WireGuard).

* **Zero Attack Surface:** No microservice is exposed via a public IP address. All services listen exclusively on the virtual `tailscale0` interface.
* **Identity-based Access:** Access permissions are managed via **Tailscale ACLs (Access Control Lists)**. An edge node in a store can only reach the specific cloud APIs required for its operation, enforcing the **Principle of Least Privilege**.
* **Seamless NAT Traversal:** Secure connectivity is maintained behind restrictive store firewalls without requiring manual network configuration or static IPs.

## 2. End-to-End Encryption (TLS)
Security does not stop at the network perimeter. All internal traffic is fully encrypted using TLS.

* **Automated PKI:** We utilize the `tailscale cert` feature. This provides each microservice with a valid, publicly trusted TLS certificate (issued via Let's Encrypt) automatically.
* **Browser-Native APIs:** This architecture ensures a **Secure Context**, enabling the use of sensitive browser features like the **Web Camera API** (for smartphone-based scanning) in-store without certificate warnings or security bypasses.

## 3. Go-Native Security (`tsnet`)
To further minimize the attack surface, we integrate networking directly into the Go binaries:
* The microservices leverage the `tsnet` library, allowing them to join the mesh in user-space.
* Services do not require **root privileges** on the host OS to listen on port 443 or manage network interfaces, adhering to **Security Best Practices**.
* Secrets (such as Tailscale Auth-Keys) are injected securely via **GCP Secret Manager** and are never stored in the source code.

## 4. SRE & Compliance
* **Audit Logs:** Every connection within the mesh is logged, providing full observability and traceability of internal traffic.
* **Automated Revocation:** If an edge node is decommissioned or compromised, its identity can be revoked instantly through the Tailscale control plane without affecting the rest of the infrastructure.
* **Vulnerability Management:** We actively monitor GitHub Security Alerts and maintain a strict **Key Rotation Policy** for all infrastructure credentials.

---