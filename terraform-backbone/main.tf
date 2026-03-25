# --- Google Cloud Provider Configuration ---

provider "google" {
  project = var.project_id
  region  = var.region # Default is europe-west3
}

# --- Artifact Registry for Retail-Edge Images ---

resource "google_artifact_registry_repository" "retail_repo" {
  location      = "europe-west3"
  repository_id = "retail-edge-repo"
  description   = "Docker Repository for Retail Edge Services"
  format        = "DOCKER"
}

# --- GKE Cluster Definition (Central Backbone) ---

resource "google_container_cluster" "retail_edge" {
  name     = "retail-edge-cluster"
  location = "europe-west3-b"

  # Minimal configuration for testing to keep costs low
  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    spot         = true # Cheaper and better for testing, not for production

    # Needed for Workload Identity and Logging
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  # Ensure the cluster is deleted cleanly
  deletion_protection = false
}

# --- Workload Identity Federation (Security Bridge for GitHub) ---

# 1. Create a Workload Identity Pool for GitHub Actions
resource "google_iam_workload_identity_pool" "github_pool" {
  workload_identity_pool_id = "github-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions to access GCP resources"
}

# 2. Configure the OIDC Provider (Updated ID to avoid name conflict)
resource "google_iam_workload_identity_pool_provider" "github_provider" {
  workload_identity_pool_id = google_iam_workload_identity_pool.github_pool.workload_identity_pool_id
  # English comment: Changed ID to bypass soft-deleted naming conflicts
  workload_identity_pool_provider_id = "github-provider-v2"

  attribute_mapping = {
    "google.subject"             = "assertion.sub"
    "attribute.actor"            = "assertion.actor"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
  }

  # English comment: Required condition for secure repository-based access
  attribute_condition = "attribute.repository == 'demirdilek/Retail-Backbone-GCP'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# 3. Dedicated Service Account for GitHub Actions
resource "google_service_account" "github_actions_sa" {
  account_id   = "github-actions-sa"
  display_name = "GitHub Actions Service Account"
}

# 4. Allow GitHub to impersonate the Service Account
resource "google_service_account_iam_member" "github_sa_user" {
  service_account_id = google_service_account.github_actions_sa.name
  role               = "roles/iam.workloadIdentityUser"

  # English comment: The member string must use the attribute defined in the mapping above
  member = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_pool.name}/attribute.repository/demirdilek/Retail-Backbone-GCP"
}

# 5. Grant the Service Account write access to the Artifact Registry
resource "google_project_iam_member" "gar_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.github_actions_sa.email}"
}

# --- Outputs for CI/CD Configuration ---

output "workload_identity_provider" {
  description = "The full name of the Workload Identity Provider for GitHub Actions"
  value       = google_iam_workload_identity_pool_provider.github_provider.name
}

output "service_account_email" {
  description = "The email of the Service Account for GitHub Actions"
  value       = google_service_account.github_actions_sa.email
}