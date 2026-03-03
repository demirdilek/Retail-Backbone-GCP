terraform {
  backend "gcs" {
    bucket  = "retail-edge-tfstate-retail-backbone-gcp" # Hier den Namen von oben einsetzen
    prefix  = "terraform/state"
  }
}