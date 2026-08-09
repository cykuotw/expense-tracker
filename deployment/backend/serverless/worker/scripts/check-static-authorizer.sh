#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAIN_TF="$(cd "$SCRIPT_DIR/../tf" && pwd)/main.tf"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_exact_line() {
  local expected="$1"
  local count
  count="$(grep -Fxc "$expected" "$MAIN_TF" || true)"
  [[ "$count" == "1" ]] || fail "expected exactly one Terraform line: $expected"
}

require_absent() {
  local forbidden="$1"
  if grep -Fq "$forbidden" "$MAIN_TF"; then
    fail "unexpected Terraform setting: $forbidden"
  fi
}

require_absent_regex() {
  local forbidden="$1"
  if grep -Eq "$forbidden" "$MAIN_TF"; then
    fail "unexpected Terraform pattern: $forbidden"
  fi
}

terraform -chdir="$(dirname "$MAIN_TF")" fmt -check >/dev/null

require_exact_line '  api_path      = "/api/v0"'
require_exact_line '  google_issuer = "https://accounts.google.com"'
require_exact_line '  authorizer_type  = "JWT"'
require_exact_line '  identity_sources = ["$request.header.Authorization"]'
require_exact_line '    audience = [var.google_client_id]'
require_exact_line '    issuer   = local.google_issuer'
require_exact_line '  route_key          = "POST ${local.api_path}/auth/google/exchange"'
require_exact_line '  authorization_type = "JWT"'
require_exact_line '  authorizer_id      = aws_apigatewayv2_authorizer.google.id'
require_exact_line '  route_key          = "$default"'
require_exact_line '  authorization_type = "NONE"'
require_exact_line '      GOOGLE_CLIENT_ID       = var.google_client_id'
require_exact_line '      GOOGLE_OAUTH_ENABLED   = "true"'
require_exact_line '      GOOGLE_EXCHANGE_MODE   = "upstream_verified"'

require_absent_regex 'issuer[[:space:]]*=[[:space:]]*"accounts\.google\.com"'
require_absent 'authorization_scopes'
require_absent 'cors_configuration'

printf 'Static Google JWT authorizer and route boundary checks passed\n'
