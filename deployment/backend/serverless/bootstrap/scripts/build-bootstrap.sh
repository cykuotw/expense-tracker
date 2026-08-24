#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOTSTRAP_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BOOTSTRAP_ROOT/../../../.." && pwd)"
BUILD_DIR="$BOOTSTRAP_ROOT/build"
ARTIFACT="$BUILD_DIR/bootstrap.zip"
STAGING_DIR="$(mktemp -d)"
STAGED_ARTIFACT="$STAGING_DIR/bootstrap.zip"

cleanup() {
  rm -rf "$STAGING_DIR"
}
trap cleanup EXIT

mkdir -p "$BUILD_DIR" "$STAGING_DIR/migrations"
(
  cd "$REPO_ROOT"
  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "$STAGING_DIR/bootstrap" \
    ./backend/cmd/bootstrap-serverless
)
cp -R "$REPO_ROOT/backend/cmd/migrate/migrations/." "$STAGING_DIR/migrations/"
chmod 0755 "$STAGING_DIR/bootstrap"
find "$STAGING_DIR" -type f -exec touch -t 198001010000 {} +
(
  cd "$STAGING_DIR"
  find bootstrap migrations -type f -print | LC_ALL=C sort | zip -q -X "$STAGED_ARTIFACT" -@
)
mv "$STAGED_ARTIFACT" "$ARTIFACT"

[[ "$(unzip -Z1 "$ARTIFACT" | head -n 1)" == "bootstrap" ]] || {
  printf 'error: bootstrap must be the first artifact entry\n' >&2
  exit 1
}
unzip -p "$ARTIFACT" bootstrap | file - | grep -q 'ARM aarch64' || {
  printf 'error: bootstrap executable is not ARM64\n' >&2
  exit 1
}
[[ "$(unzip -Z1 "$ARTIFACT" | grep -c '^migrations/.*\.up\.sql$')" -gt 0 ]] || {
  printf 'error: bootstrap artifact has no up migrations\n' >&2
  exit 1
}

printf 'Built %s\n' "$ARTIFACT"
