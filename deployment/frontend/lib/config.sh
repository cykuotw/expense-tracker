FRONTEND_ROOT="${FRONTEND_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
REPO_ROOT="${REPO_ROOT:-$(cd "$FRONTEND_ROOT/../.." && pwd)}"

TF_DIR="${TF_DIR:-$REPO_ROOT/deployment/backend/serverful/tf}"
TF_VARS_FILE="${TF_VARS_FILE:-terraform.tfvars}"
FRONTEND_DIST_DIR="${FRONTEND_DIST_DIR:-frontend/dist}"
API_URL="${API_URL:-/api/v0}"
GOOGLE_OAUTH_ENABLED="${GOOGLE_OAUTH_ENABLED:-false}"
GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID:-}"
