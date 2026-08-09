#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$WORKER_ROOT/../../../.." && pwd)"
BUILD_DIR="$WORKER_ROOT/build"
ARTIFACT="$BUILD_DIR/worker.zip"
STAGING_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$STAGING_DIR"
}
trap cleanup EXIT

mkdir -p "$BUILD_DIR"

(
  cd "$REPO_ROOT"
  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X expense-tracker/backend/config.BuildMode=release" \
    -o "$STAGING_DIR/bootstrap" \
    ./backend/cmd/tracker-serverless
)

chmod 0755 "$STAGING_DIR/bootstrap"
touch -t 198001010000 "$STAGING_DIR/bootstrap"
(
  cd "$STAGING_DIR"
  zip -q -X worker.zip bootstrap
)
mv "$STAGING_DIR/worker.zip" "$ARTIFACT"

[[ "$(unzip -Z1 "$ARTIFACT")" == "bootstrap" ]] || {
  printf 'error: worker artifact must contain only bootstrap\n' >&2
  exit 1
}
unzip -p "$ARTIFACT" bootstrap | file - | grep -q 'ARM aarch64' || {
  printf 'error: worker bootstrap is not an ARM64 executable\n' >&2
  exit 1
}

printf 'Built %s\n' "$ARTIFACT"
