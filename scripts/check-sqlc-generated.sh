#!/usr/bin/env bash
set -euo pipefail

sqlc_bin="${SQLC_BIN:-sqlc}"
if ! command -v "$sqlc_bin" >/dev/null 2>&1 && [ ! -x "$sqlc_bin" ]; then
  echo "sqlc is required; install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0" >&2
  exit 1
fi

snapshot_dir="$(mktemp -d)"
trap 'rm -rf "$snapshot_dir"' EXIT

find internal/adapters/postgres/generated -type f -name '*.go' -print0 \
  | sort -z \
  | xargs -0 sha256sum >"$snapshot_dir/before.sha256"

"$sqlc_bin" generate -f sqlc.yaml

find internal/adapters/postgres/generated -type f -name '*.go' -print0 \
  | sort -z \
  | xargs -0 sha256sum >"$snapshot_dir/after.sha256"

if ! cmp -s "$snapshot_dir/before.sha256" "$snapshot_dir/after.sha256"; then
  echo "sqlc generated output changed; review generated files and query/schema inputs" >&2
  exit 1
fi

echo "sqlc generated output: PASS"
