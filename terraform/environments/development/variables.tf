variable "project_id" {
  type        = string
  description = "The unique identifier for the GCP project for resource organization and billing."
  validation {
    condition     = length(var.project_id) > 0
    error_message = "The project_id must not be empty."
  }
}

variable "project_region" {
  type        = string
  description = "The GCP region where the resources will be deployed, impacting latency and compliance."
  validation {
    condition     = length(var.project_region) > 0
    error_message = "The project_region must be specified."
  }
}

variable "environment" {
  type        = string
  description = "The deployment environment (e.g., dev, staging, prod) for resource organization and management."
}

variable "correlator_webhook_url" {
  type        = string
  description = "Ingest URL of the deployed prj-apex-alert-correlator. Leave empty to skip creating the notification channel (alert policy is still created)."
  default     = "https://prj-apex-alert-correlator-28030170607.asia-south1.run.app/v1/alerts"
}

variable "correlator_webhook_username" {
  type        = string
  description = "Basic Auth username Cloud Monitoring sends to the correlator webhook."
  default     = "apex-monitoring"
}

variable "correlator_webhook_password" {
  type        = string
  description = "Basic Auth password Cloud Monitoring sends to the correlator webhook."
  default     = "b17f4578354deadb9ccbb153af21884cc518e07be510599fda89808bc6894da1"
  sensitive   = true
}
