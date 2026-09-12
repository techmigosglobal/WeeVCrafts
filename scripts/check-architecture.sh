#!/usr/bin/env bash
set -euo pipefail

business_paths=(internal/domain internal/application internal/ports)
forbidden='htmx|jackc/pgx|go-redis|redis/go-redis|meilisearch|aws-sdk|aws/aws-sdk|s3|razorpay'

if rg --line-number --ignore-case --glob '*.go' "$forbidden" "${business_paths[@]}"; then
  echo "forbidden provider or transport import found in business boundary" >&2
  exit 1
fi

echo "architecture import boundary: PASS"
