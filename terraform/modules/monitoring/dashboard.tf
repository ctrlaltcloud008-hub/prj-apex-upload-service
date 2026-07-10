locals {
  cloud_run_filter = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"${var.cloud_run_service_name}\""
  otel_http_filter = "resource.type = \"generic_task\" AND resource.labels.namespace = \"${var.project_id}\" AND resource.labels.job = \"${var.cloud_run_service_name}\""
}

resource "google_logging_metric" "upload_requests" {
  project = var.project_id
  name    = "upload_api_requests_total"
  filter  = "${local.cloud_run_filter} AND jsonPayload.event_type = \"audit.upload.requested\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

resource "google_logging_metric" "upload_successes" {
  project = var.project_id
  name    = "upload_api_successes_total"
  filter  = "${local.cloud_run_filter} AND jsonPayload.event_type = \"audit.upload.succeeded\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

resource "google_logging_metric" "upload_failures" {
  project = var.project_id
  name    = "upload_api_failures_total"
  filter  = "${local.cloud_run_filter} AND jsonPayload.event_type = \"audit.upload.failed\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

resource "google_logging_metric" "upload_quota_rejections" {
  project = var.project_id
  name    = "upload_api_quota_rejections_total"
  filter = format("%s AND (%s)",
    local.cloud_run_filter,
    join(" OR ", [
      "jsonPayload.event_type = \"upload.quota.concurrent_exceeded\"",
      "jsonPayload.event_type = \"upload.quota.hourly_exceeded\"",
      "jsonPayload.event_type = \"upload.quota.storage_exceeded\"",
    ]),
  )

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

resource "google_logging_metric" "upload_signed_url_failures" {
  project = var.project_id
  name    = "upload_api_signed_url_failures_total"
  filter  = "${local.cloud_run_filter} AND jsonPayload.event_type = \"upload.storage.signed_url_failed\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

resource "google_monitoring_dashboard" "upload_service" {
  project = var.project_id

  dashboard_json = jsonencode({
    displayName = "${var.service_name} performance (${var.environment})"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "Request rate"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/request_count\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
              yAxis = {
                label = "requests/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 4
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "5xx error ratio"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilterRatio = {
                    numerator = {
                      filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/request_count\" AND metric.labels.response_code_class = \"5xx\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                      }
                    }
                    denominator = {
                      filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/request_count\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                      }
                    }
                  }
                }
              }]
              yAxis = {
                label = "ratio"
                scale = "LINEAR"
              }
              thresholds = [{
                label = "alert threshold"
                value = var.error_rate_threshold
              }]
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 8
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "Cloud Run p95 latency"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/request_latencies\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_PERCENTILE_95"
                      crossSeriesReducer = "REDUCE_MEAN"
                    }
                  }
                }
              }]
              yAxis = {
                label = "latency"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 6
          height = 4
          widget = {
            title = "OTEL HTTP server duration p95"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "${local.otel_http_filter} AND metric.type = \"workload.googleapis.com/http.server.request.duration\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_PERCENTILE_95"
                      crossSeriesReducer = "REDUCE_MEAN"
                    }
                  }
                }
              }]
              yAxis = {
                label = "duration"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 6
          yPos   = 4
          width  = 3
          height = 4
          widget = {
            title = "CPU utilization"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/container/cpu/utilizations\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_MEAN"
                      crossSeriesReducer = "REDUCE_MEAN"
                    }
                  }
                }
              }]
              yAxis = {
                label = "utilization"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 9
          yPos   = 4
          width  = 3
          height = 4
          widget = {
            title = "Memory utilization"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "${local.cloud_run_filter} AND metric.type = \"run.googleapis.com/container/memory/utilizations\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_MEAN"
                      crossSeriesReducer = "REDUCE_MEAN"
                    }
                  }
                }
              }]
              yAxis = {
                label = "utilization"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 8
          width  = 6
          height = 4
          widget = {
            title = "Upload outcomes"
            xyChart = {
              dataSets = [
                {
                  plotType       = "LINE"
                  targetAxis     = "Y1"
                  legendTemplate = "requested"
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.upload_requests.name}\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                      }
                    }
                  }
                },
                {
                  plotType       = "LINE"
                  targetAxis     = "Y1"
                  legendTemplate = "succeeded"
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.upload_successes.name}\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                      }
                    }
                  }
                },
                {
                  plotType       = "LINE"
                  targetAxis     = "Y1"
                  legendTemplate = "failed"
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.upload_failures.name}\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                      }
                    }
                  }
                }
              ]
              yAxis = {
                label = "events/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 6
          yPos   = 8
          width  = 3
          height = 4
          widget = {
            title = "Quota rejections"
            xyChart = {
              dataSets = [{
                plotType = "STACKED_BAR"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.upload_quota_rejections.name}\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
              yAxis = {
                label = "rejections/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 9
          yPos   = 8
          width  = 3
          height = 4
          widget = {
            title = "Signed URL failures"
            xyChart = {
              dataSets = [{
                plotType = "STACKED_BAR"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type = \"logging.googleapis.com/user/${google_logging_metric.upload_signed_url_failures.name}\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
              yAxis = {
                label = "failures/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        }
      ]
    }
  })
}
