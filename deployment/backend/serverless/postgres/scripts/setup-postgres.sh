#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POSTGRES_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="${TF_DIR:-$POSTGRES_ROOT/tf}"
SSH_USER="${SSH_USER:-ec2-user}"
SSH_PRIVATE_KEY="${SSH_PRIVATE_KEY:-}"
POSTGRES_PASSWORD_FILE="${POSTGRES_PASSWORD_FILE:-}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in terraform ssh scp install mktemp; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

[[ -n "$SSH_PRIVATE_KEY" ]] || fail 'set SSH_PRIVATE_KEY to the local EC2 private-key path'
[[ -f "$SSH_PRIVATE_KEY" ]] || fail "SSH private key not found: $SSH_PRIVATE_KEY"
[[ -n "$POSTGRES_PASSWORD_FILE" ]] || fail 'set POSTGRES_PASSWORD_FILE to the operator-owned password file'
[[ -f "$POSTGRES_PASSWORD_FILE" ]] || fail "PostgreSQL password file not found: $POSTGRES_PASSWORD_FILE"
[[ -s "$POSTGRES_PASSWORD_FILE" ]] || fail 'PostgreSQL password file is empty'

temporary_public_ip="$(terraform -chdir="$TF_DIR" output -raw serverless_postgres_temporary_public_ipv4)"
private_ip="$(terraform -chdir="$TF_DIR" output -raw serverless_postgres_host)"
vpc_cidr="$(terraform -chdir="$TF_DIR" output -raw serverless_postgres_vpc_ipv4_cidr)"
[[ -n "$temporary_public_ip" ]] || fail 'temporary public access is disabled; enable it and apply before running setup'
[[ -n "$private_ip" ]] || fail 'missing serverless_postgres_host Terraform output'
[[ "$vpc_cidr" != "0.0.0.0/0" && "$vpc_cidr" == */* ]] || fail 'invalid VPC CIDR Terraform output'

staging_file="$(mktemp)"
remote_staging="/tmp/expense-tracker-postgres-password.$RANDOM"
cleanup() {
  rm -f "$staging_file"
}
trap cleanup EXIT
install -m 0600 "$POSTGRES_PASSWORD_FILE" "$staging_file"

ssh_options=(
  -i "$SSH_PRIVATE_KEY"
  -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new
)

printf 'Uploading temporary PostgreSQL credential staging copy...\n'
scp "${ssh_options[@]}" "$staging_file" "$SSH_USER@$temporary_public_ip:$remote_staging"

printf 'Installing and configuring native PostgreSQL 16...\n'
ssh "${ssh_options[@]}" "$SSH_USER@$temporary_public_ip" \
  "sudo VPC_CIDR='$vpc_cidr' DB_PRIVATE_IP='$private_ip' PASSWORD_STAGING='$remote_staging' bash -s" <<'REMOTE'
set -euo pipefail

PGDATA=/var/lib/pgsql/data
PASSWORD_PATH=/run/expense-tracker-postgres-password
PGPASS_PATH=/run/expense-tracker-postgres-pgpass
PASSWORD_SQL_PATH=/run/expense-tracker-postgres-password.sql

cleanup() {
  rm -f "$PASSWORD_PATH" "$PGPASS_PATH" "$PASSWORD_SQL_PATH" "${PASSWORD_STAGING:-}"
}
trap cleanup EXIT

[[ "$(uname -m)" == aarch64 ]] || { printf 'expected aarch64 host\n' >&2; exit 1; }
[[ "$VPC_CIDR" != "0.0.0.0/0" && "$VPC_CIDR" == */* ]] || { printf 'invalid VPC CIDR\n' >&2; exit 1; }
[[ "$DB_PRIVATE_IP" == *.* ]] || { printf 'invalid database private IPv4\n' >&2; exit 1; }
[[ -f "$PASSWORD_STAGING" ]] || { printf 'password staging file missing\n' >&2; exit 1; }
install -o root -g root -m 0600 "$PASSWORD_STAGING" "$PASSWORD_PATH"
rm -f "$PASSWORD_STAGING"

mapfile -t password_lines < "$PASSWORD_PATH"
[[ "${#password_lines[@]}" -eq 1 && -n "${password_lines[0]}" ]] || {
  printf 'password file must contain exactly one non-empty line\n' >&2
  exit 1
}
password="${password_lines[0]}"

dnf install -y postgresql16 postgresql16-server

postgres_binary="$(rpm -ql postgresql16-server | awk '/\/postgres$/ { postgres_binary = $0 } END { print postgres_binary }')"
[[ -x "$postgres_binary" ]] || { printf 'PostgreSQL server binary not found\n' >&2; exit 1; }
"$postgres_binary" --version | grep -Eq '^postgres \(PostgreSQL\) 16\.' || {
  printf 'installed PostgreSQL major version is not 16\n' >&2
  exit 1
}
rpm -q postgresql16 postgresql16-server

setup_binary="$(command -v postgresql-setup || true)"
[[ -x "$setup_binary" ]] || { printf 'package-provided postgresql-setup not found\n' >&2; exit 1; }

if [[ ! -f "$PGDATA/PG_VERSION" ]]; then
  if [[ -d "$PGDATA" && -n "$(find "$PGDATA" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    printf 'non-empty PGDATA without PG_VERSION; refusing to initialize\n' >&2
    exit 1
  fi
  "$setup_binary" --initdb
fi

[[ -f "$PGDATA/PG_VERSION" ]] || { printf 'PG_VERSION missing after initialization\n' >&2; exit 1; }
[[ "$(<"$PGDATA/PG_VERSION")" == 16 ]] || { printf 'PGDATA major version is not 16\n' >&2; exit 1; }

sed -i '/^# BEGIN expense-tracker MVP$/,/^# END expense-tracker MVP$/d' "$PGDATA/postgresql.conf"
cat >> "$PGDATA/postgresql.conf" <<'CONF'
# BEGIN expense-tracker MVP
listen_addresses = '*'
port = 5432
max_connections = 20
password_encryption = 'scram-sha-256'
timezone = 'UTC'
# END expense-tracker MVP
CONF

sed -i '/^# BEGIN expense-tracker MVP$/,/^# END expense-tracker MVP$/d' "$PGDATA/pg_hba.conf"
cat >> "$PGDATA/pg_hba.conf" <<HBA
# BEGIN expense-tracker MVP
host    all    all    $VPC_CIDR    scram-sha-256
# END expense-tracker MVP
HBA
chown postgres:postgres "$PGDATA/postgresql.conf" "$PGDATA/pg_hba.conf"
chmod 0600 "$PGDATA/postgresql.conf" "$PGDATA/pg_hba.conf"

systemctl enable --now postgresql.service

escaped_password="${password//\'/\'\'}"
printf "ALTER ROLE postgres WITH PASSWORD '%s';\n" "$escaped_password" > "$PASSWORD_SQL_PATH"
chown postgres:postgres "$PASSWORD_SQL_PATH"
chmod 0600 "$PASSWORD_SQL_PATH"
sudo -u postgres psql --dbname postgres --set ON_ERROR_STOP=1 --file "$PASSWORD_SQL_PATH" >/dev/null
rm -f "$PASSWORD_SQL_PATH"

systemctl restart postgresql.service
sudo -u postgres pg_isready --host "$DB_PRIVATE_IP" --dbname postgres

printf '%s:5432:postgres:postgres:%s\n' "$DB_PRIVATE_IP" "$password" > "$PGPASS_PATH"
chown postgres:postgres "$PGPASS_PATH"
chmod 0600 "$PGPASS_PATH"
sudo -u postgres env PGPASSFILE="$PGPASS_PATH" \
  psql --host "$DB_PRIVATE_IP" --username postgres --dbname postgres --set ON_ERROR_STOP=1 --command 'SELECT 1' >/dev/null

systemctl status postgresql.service --no-pager
printf 'Native PostgreSQL 16 setup complete.\n'
REMOTE

printf 'PostgreSQL setup completed through temporary public address %s. Disable temporary public access after verification.\n' "$temporary_public_ip"
