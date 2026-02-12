# Google Cloud Provider configuration
# This tells Terraform which cloud we are talking to
provider "google" {
  project = "retail-backbone-project" # We will replace this later
  region  = "europe-west3"            # Frankfurt is ideal for German retail
}

# Pub/Sub Topic: The mailbox for our warehouse events
# This is where the S25 sends the data
resource "google_pubsub_topic" "warehouse_events" {
  name = "warehouse-stock-events"

  labels = {
    environment = "dev"
    component   = "backbone"
  }
}

# A subscription for the processing service
# This represents the system that will later read the data
resource "google_pubsub_subscription" "warehouse_processor" {
  name  = "warehouse-stock-processor-sub"
  topic = google_pubsub_topic.warehouse_events.name

  # Message retention (how long the cloud keeps data if the processor is down)
  message_retention_duration = "604800s" # 7 days
}

