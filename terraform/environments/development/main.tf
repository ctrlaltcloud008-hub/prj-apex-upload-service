module "storage" {
  source         = "../../modules/storage"
  project_id     = var.project_id
  project_region = var.project_region
  environment    = var.environment
}

module "logging" {
  source         = "../../modules/logging"
  project_id     = var.project_id
  project_region = var.project_region
  environment    = var.environment
}

module "monitoring" {
  source      = "../../modules/monitoring"
  project_id  = var.project_id
  environment = var.environment

  correlator_webhook_url      = var.correlator_webhook_url
  correlator_webhook_username = var.correlator_webhook_username
  correlator_webhook_password = var.correlator_webhook_password
}
