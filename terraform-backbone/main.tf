# Artifact Registry for Retail-Edge Images
resource "google_artifact_registry_repository" "retail_repo" {
  location      = "europe-west3"
  repository_id = "retail-edge-repo"
  description   = "Docker Repository for Retail Edge Services"
  format        = "DOCKER"
}

# Google Cloud Provider configuration
provider "google" {
  project = var.project_id
  region  = "europe-west3" # Frankfurt is a good choice for Retail Edge in Germany
}

# GKE Cluster definition
resource "google_container_cluster" "retail_edge" {
  name     = "retail-edge-cluster"
  location = "europe-west3-b"

  # Minimal configuration for testing to keep costs low
  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    spot         = true # cheaper and better for testing, but not suitable for production
    
    # Needed for Workload Identity and Logging
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  # Ensure the cluster is deleted cleanly
  deletion_protection = false
}