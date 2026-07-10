resource "google_storage_bucket" "uploads" {
  project  = var.project_id
  name     = var.upload_bucket
  location = var.project_region

  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 7
    }

    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }

  force_destroy = true


  labels = {
    team        = "apex"
    environment = var.environment
    managed_by  = "terraform"
  }
}
