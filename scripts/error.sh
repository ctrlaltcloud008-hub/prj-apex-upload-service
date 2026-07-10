#!/usr/bin/env bash

set -euo pipefail

URL="${URL:-https://upload-api-28030170607.asia-south1.run.app}"
REQUEST_COUNT="${REQUEST_COUNT:-8}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-50}"
response_file=$(mktemp "${TMPDIR:-/tmp}/alert-test-response.XXXXXX")
trap 'rm -f "$response_file"' EXIT

for ((i = 1; i <= REQUEST_COUNT; i++)); do
	request_id=$(uuidgen | tr '[:upper:]' '[:lower:]')

	if ! http_status=$(curl -sS -o "$response_file" \
		-w '%{http_code}' \
		-X POST "$URL/upload" \
		-H "Authorization: Bearer alert-test-$request_id" \
		-H "X-Request-ID: $request_id" \
		-H "Content-Type: application/json" \
		-d '{"file_name":"alert-test.mp4","content_type":"video/mp4","file_size_bytes":0}'); then
		echo "Request failed before receiving an HTTP response" >&2
		exit 1
	fi

	printf '%s HTTP %s (%d/%d)\n' \
		"$(date '+%Y-%m-%dT%H:%M:%S%z')" "$http_status" "$i" "$REQUEST_COUNT"
	cat "$response_file"
	echo

	if [[ "$http_status" != "500" ]]; then
		echo "Expected HTTP 500, received HTTP $http_status; stopping alert test" >&2
		exit 1
	fi

	if ((i < REQUEST_COUNT)); then
		sleep "$INTERVAL_SECONDS"
	fi
done
