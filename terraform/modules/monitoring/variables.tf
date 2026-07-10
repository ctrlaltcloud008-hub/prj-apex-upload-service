variable "project_id" {
  type        = string
  description = "GCP project ID"

  validation {
    condition     = length(var.project_id) > 0
    error_message = "project_id must not be empty."
  }
}

variable "environment" {
  type        = string
  description = "Deployment environment (e.g. dev, staging, prod)"
}

variable "service_name" {
  type        = string
  description = "Logical apex service name, stamped onto the alert as a user label so the correlator maps the incident to the right service (not the Cloud Run name)."
  default     = "prj-apex-upload-service"
}

variable "cloud_run_service_name" {
  type        = string
  description = "The deployed Cloud Run service name to filter metrics on (resource.labels.service_name)."
  default     = "upload-api"
}

variable "error_rate_threshold" {
  type        = number
  description = "5xx-to-total request ratio that trips the alert (0.05 = 5%)."
  default     = 0.05
}

variable "alert_duration" {
  type        = string
  description = "How long the ratio must stay above threshold before the alert fires."
  default     = "300s"
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
