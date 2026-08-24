#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOTSTRAP_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="${TF_DIR:-$BOOTSTRAP_ROOT/tf}"
ARTIFACT="$BOOTSTRAP_ROOT/build/bootstrap.zip"

"$SCRIPT_DIR/build-bootstrap.sh"
function_name="$(terraform -chdir="$TF_DIR" output -raw bootstrap_function_name)"
aws lambda update-function-code \
  --function-name "$function_name" \
  --zip-file "fileb://$ARTIFACT" \
  --architectures arm64 \
  >/dev/null
aws lambda wait function-updated --function-name "$function_name"
printf 'Updated bootstrap code for %s\n' "$function_name"
