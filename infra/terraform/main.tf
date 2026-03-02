# Google Cloud Provider configuration
provider "google" {
  project = var.project_id
  region  = "europe-west3" # Frankfurt is a good choice for Retail Edge in Germany
}

# GKE Cluster definition
resource "google_container_cluster" "retail_edge" {
  name     = "retail-edge-cluster"
  location = "europe-west3-a"

  # Minimal configuration for testing to keep costs low
  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    
    # Needed for Workload Identity and Logging
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  # Ensure the cluster is deleted cleanly
  deletion_protection = false
}

variable "project_id" {
  description = "The GCP Project ID"
  type        = string
}