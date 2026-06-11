terraform {
  backend "gcs" {
    bucket = "apex-upload-service-tf-state"
    prefix = "terraform/state/development"
  }
}
