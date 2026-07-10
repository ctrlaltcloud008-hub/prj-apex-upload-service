locals {
  common_labels = {
    team        = "apex"
    environment = var.environment
    managed_by  = "terraform"
  }
}

# Webhook channel → the alert-correlator ingest endpoint. Created only when a
# correlator URL is supplied (so `terraform apply` works before the correlator is
# deployed — the alert policy is still created, it just has no channel yet).
resource "google_monitoring_notification_channel" "correlator" {
  count        = var.correlator_webhook_url != "" ? 1 : 0
  project      = var.project_id
  display_name = "apex-alert-correlator (${var.environment})"
  type         = "webhook_basicauth"

  labels = {
    url      = var.correlator_webhook_url
    username = var.correlator_webhook_username
  }

  sensitive_labels {
    password = var.correlator_webhook_password
  }

  user_labels = local.common_labels
}

# High 5xx error-rate alert on the upload service's Cloud Run revision. The
# condition is a ratio: sum(5xx request rate) / sum(total request rate).
resource "google_monitoring_alert_policy" "upload_high_5xx" {
  project      = var.project_id
  display_name = "${var.service_name} high 5xx error rate"
  combiner     = "OR"
  severity     = "CRITICAL"

  conditions {
    display_name = "5xx error_rate > ${var.error_rate_threshold}"

    condition_threshold {
      filter = join(" AND ", [
        "resource.type = \"cloud_run_revision\"",
        "resource.labels.service_name = \"${var.cloud_run_service_name}\"",
        "metric.type = \"run.googleapis.com/request_count\"",
        "metric.labels.response_code_class = \"5xx\"",
      ])

      denominator_filter = join(" AND ", [
        "resource.type = \"cloud_run_revision\"",
        "resource.labels.service_name = \"${var.cloud_run_service_name}\"",
        "metric.type = \"run.googleapis.com/request_count\"",
      ])

      comparison      = "COMPARISON_GT"
      threshold_value = var.error_rate_threshold
      duration        = var.alert_duration

      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }

      denominator_aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }

      trigger {
        count = 1
      }
    }
  }

  notification_channels = google_monitoring_notification_channel.correlator[*].id

  # These land in incident.policy_user_labels in the webhook payload. service_name
  # is the logical apex name (prj-apex-upload-service) — NOT the Cloud Run name
  # (upload-api) the filter targets — so the correlator maps it to the right service.
  user_labels = merge(local.common_labels, {
    service_name = var.service_name
  })

  documentation {
    content   = "5xx error rate for ${var.service_name} (Cloud Run: ${var.cloud_run_service_name}) exceeded ${var.error_rate_threshold} for ${var.alert_duration}. Auto-investigated by prj-apex-alert-correlator."
    mime_type = "text/markdown"
  }
}
