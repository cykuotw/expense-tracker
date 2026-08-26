#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init release context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

require_root

RELEASE_ROOT_INPUT="${1:-${RELEASE_ROOT:-}}"
if [[ -z "$RELEASE_ROOT_INPUT" ]]; then
  fail "usage: deploy-edge-release.sh <release-root>"
  exit 1
fi

RELEASE_ROOT="$(cd "$RELEASE_ROOT_INPUT" && pwd)"
RUNTIME_ROOT="$RELEASE_ROOT/runtime"
DEPLOY_ROOT="$RELEASE_ROOT/deploy"
RELEASE_MANIFEST_PATH="$DEPLOY_ROOT/release-manifest.env"

if [[ ! -d "$RUNTIME_ROOT" ]]; then
  fail "runtime payload not found: $RUNTIME_ROOT"
  exit 1
fi
require_file "$RELEASE_MANIFEST_PATH"

source "$RELEASE_MANIFEST_PATH"

: "${CERTBOT_EMAIL:=}"

for key in \
  AWS_REGION \
  API_FQDN \
  CERTBOT_ENABLED \
  CERTBOT_STAGING
do
  require_env "$key"
done

if [[ "$CERTBOT_ENABLED" == "true" && -z "$CERTBOT_EMAIL" ]]; then
  fail "CERTBOT_EMAIL must be set when CERTBOT_ENABLED=true"
  exit 1
fi

CERTBOT_STAGING_FLAG=""
if [[ "$CERTBOT_STAGING" == "true" ]]; then
  CERTBOT_STAGING_FLAG="--staging"
fi

# ------------
# Install edge payload
# ------------

step 'Installing edge packages'
install_edge_packages

step 'Installing nginx configuration'
require_file "$RUNTIME_ROOT/nginx/expense-tracker.conf"
rm -f /etc/nginx/conf.d/default.conf
cp "$RUNTIME_ROOT/nginx/expense-tracker.conf" /etc/nginx/conf.d/expense-tracker.conf
nginx -t

# ------------
# Restart edge services
# ------------

step 'Restarting nginx'
systemctl enable nginx
systemctl restart nginx
systemctl status nginx --no-pager

# ------------
# Ensure certbot certificate
# ------------

if [[ "$CERTBOT_ENABLED" == "true" ]]; then
  step "Ensuring certbot certificate for $API_FQDN"
  mkdir -p /etc/letsencrypt/renewal-hooks/deploy
  cat <<'EOF_CERTBOT_HOOK' > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
#!/usr/bin/env bash
set -euo pipefail
systemctl reload nginx
EOF_CERTBOT_HOOK
  chmod 755 /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    systemctl enable --now certbot-renew.timer
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    systemctl enable --now certbot.timer
  fi
  if [[ "$CERTBOT_STAGING" != "true" ]] && certbot certificates --cert-name "$API_FQDN" 2>/dev/null | grep -q 'INVALID: TEST_CERT'; then
    certbot delete --non-interactive --cert-name "$API_FQDN"
  fi
  certbot --nginx --non-interactive --agree-tos --email "$CERTBOT_EMAIL" -d "$API_FQDN" --redirect $CERTBOT_STAGING_FLAG
  systemctl reload nginx
  systemctl status nginx --no-pager
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    systemctl status certbot-renew.timer --no-pager
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    systemctl status certbot.timer --no-pager
  fi
fi
