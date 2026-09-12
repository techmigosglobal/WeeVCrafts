#!/usr/bin/env sh
set -eu

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 is required; install it outside the production image before running this check" >&2
  exit 2
fi

BASE_URL="${BASE_URL:-http://localhost:8080}" k6 run scripts/load-smoke.js
