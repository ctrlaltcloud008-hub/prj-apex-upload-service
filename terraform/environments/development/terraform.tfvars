project_id     = "apex-494315"
project_region = "asia-south1"
environment    = "development"

# After the alert-correlator is deployed, set these to attach the webhook channel.
# Prefer passing the password via TF_VAR_correlator_webhook_password rather than
# committing it here.
# correlator_webhook_url      = "https://<correlator-cloud-run-url>/v1/alerts"
# correlator_webhook_username = "cm"
# correlator_webhook_password = "..."  # better: export TF_VAR_correlator_webhook_password
