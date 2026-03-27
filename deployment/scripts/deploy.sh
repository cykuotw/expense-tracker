#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init wrapper context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ------------
# CLI helpers
# ------------

usage() {
  cat <<'EOF'
Usage: ./deployment/scripts/deploy.sh [command]

Commands:
  all       Apply infrastructure, then deploy backend, then deploy frontend, then deploy edge.
  app       Deploy backend, then deploy frontend.
  infra     Apply Terraform-managed infrastructure only.
  frontend  Build and publish frontend assets only.
  backend   Build and release backend only.
  edge      Build and release nginx/certbot edge only.
  help      Show this help text.

Default:
  With no command, deploy.sh runs "app".
EOF
}

run_script() {
  local script_name="$1"
  "$SCRIPT_DIR/$script_name"
}

# ------------
# Parse CLI input
# ------------

COMMAND="${1:-app}"

if (( $# > 1 )); then
  usage >&2
  exit 1
fi

# ------------
# Dispatch deploy command
# ------------

case "$COMMAND" in
  all)
    run_script apply-infra.sh
    run_script deploy-backend.sh
    run_script deploy-frontend.sh
    run_script deploy-edge.sh
    ;;
  app)
    run_script deploy-backend.sh
    run_script deploy-frontend.sh
    ;;
  infra)
    run_script apply-infra.sh
    ;;
  frontend)
    run_script deploy-frontend.sh
    ;;
  backend)
    run_script deploy-backend.sh
    ;;
  edge)
    run_script deploy-edge.sh
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
