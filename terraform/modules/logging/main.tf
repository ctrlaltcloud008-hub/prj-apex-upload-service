resource "google_bigquery_dataset" "audit_logs" {
  project    = var.project_id
  dataset_id = var.dataset_id
  location   = var.project_region

  default_table_expiration_ms = 365 * 24 * 60 * 60 * 1000
  delete_contents_on_destroy  = true

  labels = {
    team        = "apex"
    environment = var.environment
    managed_by  = "terraform"
  }
}

# View over the daily tables Cloud Logging auto-creates (run_googleapis_com_stdout_YYYYMMDD).
# The wildcard covers all existing and future daily partitions.
resource "google_bigquery_table" "audit_events" {
  count               = var.create_audit_view ? 1 : 0
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.audit_logs.dataset_id
  table_id            = var.table_id
  deletion_protection = false

  # The view validates against run_googleapis_com_stdout_* at creation time, so
  # the sink (and at least one routed log) must exist first.
  depends_on = [google_logging_project_sink.audit_logs]

  view {
    use_legacy_sql = false
    query          = <<-EOQ
      SELECT
        timestamp,
        receiveTimestamp,
        severity,
        logName,
        insertId,
        trace,
        spanId,
        traceSampled,
        resource.labels.service_name   AS service_name,
        resource.labels.location       AS location,
        jsonPayload.event_type         AS event_type,
        jsonPayload.msg                AS message,
        jsonPayload.level              AS level,
        jsonPayload.actor              AS actor,
        jsonPayload.service            AS service,
        jsonPayload.app_env            AS app_env,
        jsonPayload.region             AS region,
        jsonPayload.trace_id           AS trace_id,
        jsonPayload.span_id            AS span_id,
        jsonPayload.request_id         AS request_id,
        jsonPayload.user_id            AS user_id,
        jsonPayload.user_tier          AS user_tier,
        jsonPayload.client_region_hint AS client_region_hint,
        jsonPayload.method             AS method,
        jsonPayload.path               AS path,
        jsonPayload.file_name          AS file_name,
        jsonPayload.file_size_bytes    AS file_size_bytes,
        jsonPayload.content_type       AS content_type,
        jsonPayload.video_id           AS video_id,
        jsonPayload.error_reason       AS error_reason,
        jsonPayload.error              AS error
      FROM `${var.project_id}.${var.dataset_id}.run_googleapis_com_stdout_*`
    EOQ
  }

  labels = {
    team        = "apex"
    environment = var.environment
    managed_by  = "terraform"
  }
}

resource "google_logging_project_sink" "audit_logs" {
  project                = var.project_id
  name                   = "apex-audit-logs-sink"
  destination            = "bigquery.googleapis.com/projects/${var.project_id}/datasets/${google_bigquery_dataset.audit_logs.dataset_id}"
  filter                 = "jsonPayload.audit = true"
  unique_writer_identity = true
}

resource "google_project_iam_member" "audit_logs_sink_writer" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = google_logging_project_sink.audit_logs.writer_identity
}
