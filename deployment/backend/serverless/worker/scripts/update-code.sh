#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="${TF_DIR:-$WORKER_ROOT/tf}"
ARTIFACT="$WORKER_ROOT/build/worker.zip"

"$SCRIPT_DIR/build-worker.sh"
function_name="$(terraform -chdir="$TF_DIR" output -raw worker_function_name)"

aws lambda update-function-code \
  --function-name "$function_name" \
  --zip-file "fileb://$ARTIFACT" \
  --architectures arm64 \
  >/dev/null

aws lambda wait function-updated --function-name "$function_name"
printf 'Updated worker code for %s\n' "$function_name"
