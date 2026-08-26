from __future__ import annotations

import os
import shlex
import tempfile
import time
import uuid
from pathlib import Path
from typing import Any

from common.command import CommandError, run
from config import Config


REMOTE_SETUP = r'''#!/usr/bin/env bash
set -euo pipefail

PGDATA=/var/lib/pgsql/data
PASSWORD_PATH=/run/expense-tracker-postgres-password
PGPASS_PATH=/run/expense-tracker-postgres-pgpass
PASSWORD_SQL_PATH=/run/expense-tracker-postgres-password.sql

cleanup() {
  rm -f "$PASSWORD_PATH" "$PGPASS_PATH" "$PASSWORD_SQL_PATH" "${PASSWORD_STAGING:-}" "$0"
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

dnf install -y postgresql16 postgresql16-server postgresql16-contrib
postgres_binary="$(rpm -ql postgresql16-server | awk '/\/postgres$/ { value = $0 } END { print value }')"
[[ -x "$postgres_binary" ]] || { printf 'PostgreSQL server binary not found\n' >&2; exit 1; }
"$postgres_binary" --version | grep -Eq '^postgres \(PostgreSQL\) 16\.'
pgcrypto_control="$(rpm -ql postgresql16-contrib | awk '/\/extension\/pgcrypto\.control$/ { value = $0 } END { print value }')"
[[ -f "$pgcrypto_control" ]] || { printf 'pgcrypto extension control file not found\n' >&2; exit 1; }
setup_binary="$(command -v postgresql-setup || true)"
[[ -x "$setup_binary" ]] || { printf 'postgresql-setup not found\n' >&2; exit 1; }

if [[ ! -f "$PGDATA/PG_VERSION" ]]; then
  if [[ -d "$PGDATA" && -n "$(find "$PGDATA" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    printf 'non-empty PGDATA without PG_VERSION; refusing to initialize\n' >&2
    exit 1
  fi
  "$setup_binary" --initdb
fi
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
sudo -u postgres psql --dbname postgres --set ON_ERROR_STOP=1 --tuples-only --no-align \
  --command "SELECT 1 FROM pg_available_extensions WHERE name = 'pgcrypto'" | grep -qx 1
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
printf 'PostgreSQL 16 setup and authenticated private readiness passed.\n'
'''


def setup(config: Config, outputs: dict[str, Any]) -> None:
    public_ip = str(outputs.get("database_temporary_public_ipv4", ""))
    private_ip = str(outputs.get("database_host", ""))
    vpc_cidr = str(outputs.get("vpc_ipv4_cidr", ""))
    if not public_ip or not private_ip or not vpc_cidr:
        raise CommandError("database setup requires temporary public IP, private IP, and VPC CIDR outputs")
    key = config.local_credentials.ssh_private_key_file
    suffix = uuid.uuid4().hex
    remote_script = f"/tmp/expense-tracker-postgres-setup-{suffix}.sh"
    remote_password = f"/tmp/expense-tracker-postgres-password-{suffix}"
    with tempfile.TemporaryDirectory(prefix="expense-postgres-setup-") as temporary:
        root = Path(temporary)
        script = root / "setup.sh"
        password = root / "password"
        known_hosts = root / "known_hosts"
        script.write_text(REMOTE_SETUP)
        password.write_text(config.database.admin_password + "\n")
        os.chmod(script, 0o700)
        os.chmod(password, 0o600)
        scanned_output = ""
        for _ in range(30):
            scanned = run(["ssh-keyscan", "-T", "5", public_ip], check=False)
            scanned_output = scanned.stdout
            if scanned.returncode == 0 and scanned_output.strip():
                break
            time.sleep(5)
        if not scanned_output.strip():
            raise CommandError("PostgreSQL host did not become reachable for SSH")
        known_hosts.write_text(scanned_output)
        os.chmod(known_hosts, 0o600)
        options = ["-i", str(key), "-o", "BatchMode=yes", "-o", f"UserKnownHostsFile={known_hosts}", "-o", "StrictHostKeyChecking=yes"]
        destination = f"ec2-user@{public_ip}"
        run(["scp", *options, str(script), f"{destination}:{remote_script}"], capture=False)
        run(["scp", *options, str(password), f"{destination}:{remote_password}"], capture=False)
        environment = " ".join([
            f"VPC_CIDR={shlex.quote(vpc_cidr)}",
            f"DB_PRIVATE_IP={shlex.quote(private_ip)}",
            f"PASSWORD_STAGING={shlex.quote(remote_password)}",
        ])
        run(["ssh", *options, destination, f"sudo env {environment} bash {shlex.quote(remote_script)}"], capture=False)
