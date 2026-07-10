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

variable "create_audit_view" {
  type        = bool
  description = <<-EOD
    Whether to create the audit_events view. The view reads from the
    run_googleapis_com_stdout_* wildcard, which BigQuery validates at creation
    time and which only matches once Cloud Logging has routed at least one log
    into the dataset. Leave false on the first apply (dataset + sink only); set
    true on a later apply after audit logs have started flowing.
  EOD
  default     = false
}
