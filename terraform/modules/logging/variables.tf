variable "project_id" {
  type        = string
  description = "GCP project ID"

  validation {
    condition     = length(var.project_id) > 0
    error_message = "project_id must not be empty."
  }
}

variable "project_region" {
  type        = string
  description = "GCP region for the BigQuery dataset"

  validation {
    condition     = length(var.project_region) > 0
    error_message = "project_region must not be empty."
  }
}

variable "environment" {
  type        = string
  description = "Deployment environment (e.g. dev, staging, prod)"
}

variable "dataset_id" {
  type        = string
  description = "BigQuery dataset ID for audit logs"
  default     = "apex_audit_logs"
}

variable "table_id" {
  type        = string
  description = "BigQuery table ID for the audit events sink"
  default     = "audit_events"
}
