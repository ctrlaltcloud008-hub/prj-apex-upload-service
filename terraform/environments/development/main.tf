module "storage" {
  source         = "../../modules/storage"
  project_id     = var.project_id
  project_region = var.project_region
  environment    = var.environment
}
