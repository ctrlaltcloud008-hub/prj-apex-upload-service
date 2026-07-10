output "alert_policy_id" {
  description = "Full resource ID of the 5xx error-rate alert policy"
  value       = google_monitoring_alert_policy.upload_high_5xx.id
}

output "alert_policy_name" {
  description = "Cloud Monitoring name of the alert policy"
  value       = google_monitoring_alert_policy.upload_high_5xx.name
}

output "notification_channel_id" {
  description = "ID of the correlator webhook channel (null until correlator_webhook_url is set)"
  value       = one(google_monitoring_notification_channel.correlator[*].id)
}

output "dashboard_id" {
  description = "Full resource ID of the upload-service performance dashboard"
  value       = google_monitoring_dashboard.upload_service.id
}
