output "dataset_id" {
  description = "BigQuery dataset ID for audit logs"
  value       = google_bigquery_dataset.audit_logs.dataset_id
}

output "table_id" {
  description = "BigQuery table ID for audit events"
  value       = google_bigquery_table.audit_events.table_id
}

output "sink_name" {
  description = "Name of the Cloud Logging project sink"
  value       = google_logging_project_sink.audit_logs.name
}

output "sink_writer_identity" {
  description = "Service account identity used by the sink to write to BigQuery"
  value       = google_logging_project_sink.audit_logs.writer_identity
}
