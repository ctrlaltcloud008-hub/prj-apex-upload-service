# --- Variables ---

project_id  := "apex-494315"
region      := "asia-south1"
registry    := "asia-south1-docker.pkg.dev"
repo        := "prj-apex-artifact-registry"
service     := "upload-api"
sa          := "ci-cd-98@apex-494315.iam.gserviceaccount.com"
image       := registry + "/" + project_id + "/" + repo + "/" + service
git_sha       := `git rev-parse --short HEAD`
vpc_connector := "projects/apex-494315/locations/asia-south1/connectors/apex-vpc-conn-development"

spanner_db    := "projects/apex-494315/instances/apex-spanner-instance/databases/apex-database"
upload_bucket := "prj-apex-upload-bucket"

# --- Local Dev ---

up:
  docker compose up -d
  docker compose wait spanner-init

down:
  docker compose down

run:
  docker run -d \
    -v $HOME/.config/gcloud:/tmp/gcloud:ro \
    -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcloud/application_default_credentials.json \
    -e OTEL_RESOURCE_ATTRIBUTES="gcp.project_id={{project_id}}" \
    -e OTEL_EXPORTER_OTLP_ENDPOINT=https://telemetry.googleapis.com \
    -e GOOGLE_CLOUD_QUOTA_PROJECT="{{project_id}}" \
    --name prj-apex-upload-service \
    --network prj-apex-upload-service_spanner-net \
    -p 8000:8000 \
    -e SPANNER_EMULATOR_HOST=spanner-emulator:9010 \
    -e REDIS_ADDR=redis:6379 \
    -e APP_ENV=local \
    prj-apex-upload-service

# --- Build & Push ---

docker-auth:
  gcloud auth configure-docker {{registry}}

build:
  docker buildx build --platform=linux/amd64 --load \
    -t prj-apex-upload-service \
    .

push:
  docker buildx build --platform=linux/amd64 --push \
    -t {{image}}:{{git_sha}} \
    -t {{image}}:latest \
    .

# --- Cloud Run Deploy (temporary — migrate to GKE later) ---

deploy: push
  #!/usr/bin/env bash
  set -euo pipefail
  redis_host=$(gcloud redis instances describe apex-redis-development \
    --region={{region}} --project={{project_id}} --format='value(host)')
  buckets_json='{"{{region}}":"{{upload_bucket}}"}'
  gcloud run deploy {{service}} \
    --image={{image}}:{{git_sha}} \
    --region={{region}} \
    --platform=managed \
    --allow-unauthenticated \
    --service-account={{sa}} \
    --memory=512Mi --cpu=1 \
    --min-instances=0 --max-instances=10 \
    --concurrency=80 --timeout=300 \
    --vpc-connector={{vpc_connector}} \
    --vpc-egress=private-ranges-only \
    --clear-secrets \
    --set-env-vars="^|^APP_ENV=development|SERVICE={{service}}|REGION={{region}}|PROJECT_ID={{project_id}}|SPANNER_DATABASE={{spanner_db}}|OTEL_RESOURCE_ATTRIBUTES=gcp.project_id={{project_id}}|OTEL_EXPORTER_OTLP_ENDPOINT=https://telemetry.googleapis.com|GOOGLE_CLOUD_QUOTA_PROJECT={{project_id}}|REDIS_ADDR=${redis_host}:6379|BUCKETS_JSON=${buckets_json}"


migrate-up:
  export SPANNER_EMULATOR_HOST=localhost:9010 && \
  export SPANNER_PROJECT_ID=test-project && \
  export SPANNER_INSTANCE_ID=test-instance && \
  export SPANNER_DATABASE_ID=test-database && \
  wrench migrate up --directory schema

migrate-repair:
  wrench migrate repair \
    --project {{project_id}} \
    --instance apex-spanner-instance \
    --database apex-database \
    --directory schema

spanner-cli:
  SPANNER_EMULATOR_HOST=localhost:9010 spanner-cli sql \
    --project test-project \
    --instance test-instance \
    --database test-database

# --- Code Quality ---

fmt:
  go fmt ./...

vet:
  go vet ./...

lint:
  golangci-lint run

test:
  go test ./... -v

# --- Test Requests ---

upload:
  curl -X POST  https://upload-api-28030170607.asia-south1.run.app/upload \
    -H "Authorization: Bearer apex" \
    -H "Content-Type: application/json" \
    -H "X-Client-Region: asia-south1" \
    -d '{"file_name":"sample-video.mp4","content_type":"video/mp4","file_size_bytes":10485760}'
