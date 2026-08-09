#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${TF_DIR:-$(cd "$SCRIPT_DIR/../tf" && pwd)}"
api_endpoint="$(terraform -chdir="$TF_DIR" output -raw worker_api_endpoint)"

curl -fsS "${api_endpoint%/}/api/v0/health"
printf '\n'
