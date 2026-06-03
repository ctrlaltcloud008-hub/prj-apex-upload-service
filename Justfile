build:
  docker buildx build --no-cache --platform=linux/amd64 -t prj-apex-upload-service .

network:
  docker network create spanner-net

run:
  docker run -d -v $HOME/.config/gcloud:/tmp/gcloud:ro -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcloud/application_default_credentials.json -e OTEL_RESOURCE_ATTRIBUTES="gcp.project_id=apex-494315" -e OTEL_EXPORTER_OTLP_ENDPOINT=https://telemetry.googleapis.com -e GOOGLE_CLOUD_QUOTA_PROJECT="apex-494315" --name prj-apex-upload-service --network spanner-net -p 8000:8000 -e SPANNER_EMULATOR_HOST=spanner-emulator:9010 -e APP_ENV=local prj-apex-upload-service

docker-auth:
	gcloud auth configure-docker asia-south1-docker.pkg.dev

docker-tag:
  docker tag prj-apex-upload-service asia-south1-docker.pkg.dev/apex-494315/prj-apex-artifact-registry/upload-api:latest

docker-push: docker-tag
  docker push asia-south1-docker.pkg.dev/apex-494315/prj-apex-artifact-registry/upload-api:latest

spanner-up:
  docker run --name spanner-emulator --network spanner-net -p 9010:9010 -p 9020:9020 -d gcr.io/cloud-spanner-emulator/emulator

migrate-up:
  export SPANNER_EMULATOR_HOST=localhost:9010 && export SPANNER_PROJECT_ID=test-project && export SPANNER_INSTANCE_ID=test-instance && export SPANNER_DATABASE_ID=test-database  && wrench migrate up --directory schema

migrate-gcp:
	wrench migrate up \
		--project apex-494315 \
		--instance apex-spanner-instance \
		--database apex-database \
		--directory schema

emulator-config:
  gcloud config configurations create emulator 2>/dev/null || true
  gcloud config configurations activate emulator
  gcloud config set auth/disable_credentials true
  gcloud config set project test-project
  gcloud config set api_endpoint_overrides/spanner http://localhost:9020/

spanner-create:
  gcloud spanner instances create test-instance --config=regional-us-central1 --description="Local Instance" --nodes=1
  gcloud spanner databases create test-database --instance test-instance

spanner-cli:
  SPANNER_EMULATOR_HOST=localhost:9010 spanner-cli sql --project test-project --instance test-instance --database test-database

fmt:
  go fmt ./...

vet:
  go vet ./...

lint:
  golangci-lint run

test:
  go test ./... -v

upload:
  curl -X POST https://upload-api-28030170607.asia-south1.run.app/upload \
    -H "Authorization: Bearer any-token" \
    -H "Content-Type: application/json" \
    -H "X-Client-Region: us-east1" \
    -d '{"file_name":"sample-video.mp4","content_type":"video/mp4","file_size_bytes":10485760}'


upload_env_vars := "APP_ENV=development,SERVICE=upload-api,REGION=asia-south1,PROJECT_ID=apex-494315,SPANNER_DATABASE=projects/apex-494315/instances/apex-spanner-instance/databases/apex-database,BUCKETS_JSON={\"asia-south1\":\"apex-494315-development-uploads-asia-south1\"},OTEL_RESOURCE_ATTRIBUTES=gcp.project_id=apex-494315,OTEL_EXPORTER_OTLP_ENDPOINT=https://telemetry.googleapis.com,GOOGLE_CLOUD_QUOTA_PROJECT=apex-494315"

deploy: docker-push
	gcloud run deploy upload-api \
		--image=asia-south1-docker.pkg.dev/apex-494315/prj-apex-artifact-registry/upload-api:latest \
		--region=asia-south1 \
		--platform=managed \
		--allow-unauthenticated \
		--service-account=ci-cd-98@apex-494315.iam.gserviceaccount.com \
		--memory=512Mi --cpu=1 \
		--min-instances=0 --max-instances=10 \
		--concurrency=80 --timeout=300 \
		--clear-secrets && \
	gcloud run deploy upload-api \
		--image=asia-south1-docker.pkg.dev/apex-494315/prj-apex-artifact-registry/upload-api:latest \
		--region=asia-south1 \
		--platform=managed \
		--allow-unauthenticated \
		--service-account=ci-cd-98@apex-494315.iam.gserviceaccount.com \
		--memory=512Mi --cpu=1 \
		--min-instances=0 --max-instances=10 \
		--concurrency=80 --timeout=300 \
		--set-env-vars='{{upload_env_vars}}'
