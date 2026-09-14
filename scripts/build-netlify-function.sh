#!/usr/bin/env bash
# Build the Netlify Functions Go binary for the WeeVCrafts preview.
#
# Netlify's build image provides Go when GO_VERSION is set in netlify.toml.
# The build emits a binary named after the function ("web") into the
# functions directory configured in netlify.toml.
set -euo pipefail

cd "$(dirname "$0")/.."

output_dir="netlify/functions-dist"
mkdir -p "$output_dir" netlify/publish

# Keep the publish directory non-empty with a marker page; Netlify requires
# a publish directory even though a catch-all rewrite serves all traffic.
if [ ! -e netlify/publish/index.html ]; then
  cat > netlify/publish/index.html <<'HTML'
<!doctype html>
<title>WeeVCrafts preview</title>
<p>This deploy serves the WeeVCrafts UI preview through a Netlify Function.</p>
HTML
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
  -o "$output_dir/web" ./netlify/functions/web
