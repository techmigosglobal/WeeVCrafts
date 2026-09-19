#!/usr/bin/env bash
# Build the real application and bounded maintenance Netlify Go Functions.
set -euo pipefail

cd "$(dirname "$0")/.."

output_dir="netlify/functions-dist"
mkdir -p "$output_dir"

for function in web maintenance; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
    -o "$output_dir/$function" "./netlify/functions/$function"
done
