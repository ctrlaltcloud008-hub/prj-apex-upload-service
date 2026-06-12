terraform {
  backend "gcs" {
    bucket = "apex-bkt-tf-state"
    prefix = "terraform/state/apex-upload-service/development"
  }
}
