# Remote backend to store Terraform state in Google Cloud Storage
terraform {
  backend "gcs" {
    bucket = "retail-backbone-tf-state" # Change this to a unique bucket name you created
    prefix = "terraform/state"
  }
}