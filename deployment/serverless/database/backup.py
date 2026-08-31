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


REMOTE_BACKUP_CONFIGURATION = r'''#!/usr/bin/env bash
set -euo pipefail

cleanup() {
  rm -f "$0"
}
trap cleanup EXIT

[[ "$BACKUP_BUCKET" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]] || {
  printf 'invalid backup bucket name\n' >&2
  exit 1
}
[[ "$BACKUP_REGION" =~ ^[a-z]{2}(-[a-z]+)+-[0-9]+$ ]] || {
  printf 'invalid backup region\n' >&2
  exit 1
}
[[ "$BACKUP_DATABASE" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] || {
  printf 'invalid backup database name\n' >&2
  exit 1
}
[[ "$BACKUP_TIME" =~ ^([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$ ]] || {
  printf 'invalid backup time\n' >&2
  exit 1
}
[[ "$BACKUP_TIMEZONE" =~ ^(UTC|[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)+)$ ]] || {
  printf 'invalid backup time zone\n' >&2
  exit 1
}
systemd-analyze calendar "*-*-* ${BACKUP_TIME} ${BACKUP_TIMEZONE}" >/dev/null || {
  printf 'invalid systemd backup calendar\n' >&2
  exit 1
}
if ! command -v aws >/dev/null; then
  dnf install -y awscli2
fi
command -v aws >/dev/null
command -v pg_dump >/dev/null
install -d -o postgres -g postgres -m 0700 /var/lib/expense-tracker-backups
chown postgres:postgres /var/lib/expense-tracker-backups
chmod 0700 /var/lib/expense-tracker-backups

cat > /etc/expense-tracker-postgres-backup.env <<EOF
BACKUP_BUCKET=$BACKUP_BUCKET
BACKUP_REGION=$BACKUP_REGION
BACKUP_DATABASE=$BACKUP_DATABASE
EOF
chown root:root /etc/expense-tracker-postgres-backup.env
chmod 0600 /etc/expense-tracker-postgres-backup.env

cat > /usr/local/sbin/expense-tracker-postgres-backup <<'BACKUP'
#!/usr/bin/env bash
set -euo pipefail
umask 077

source /etc/expense-tracker-postgres-backup.env
exec 9>/var/lock/expense-tracker-postgres-backup.lock
flock -n 9 || exit 0

archive="$(mktemp /var/lib/expense-tracker-backups/${BACKUP_DATABASE}.XXXXXXXX.dump)"
status_report="$(mktemp /var/lib/expense-tracker-backups/${BACKUP_DATABASE}-status.XXXXXXXX.txt)"
cleanup() {
  rm -f "$archive" "$status_report"
}
trap cleanup EXIT

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
key="daily/${BACKUP_DATABASE}-${timestamp}.dump"
chown postgres:postgres "$archive"
sudo -u postgres pg_dump --dbname="$BACKUP_DATABASE" --format=custom --file="$archive"
test -s "$archive"
aws s3 cp "$archive" "s3://${BACKUP_BUCKET}/${key}" --region "$BACKUP_REGION" --sse AES256 --only-show-errors
printf 'status=success\ncompleted_at=%s\nbackup_key=%s\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$key" > "$status_report"
aws s3 cp "$status_report" "s3://${BACKUP_BUCKET}/status/latest.txt" --region "$BACKUP_REGION" --sse AES256 --only-show-errors
printf 'PostgreSQL logical backup uploaded: s3://%s/%s\n' "$BACKUP_BUCKET" "$key"
BACKUP
chown root:root /usr/local/sbin/expense-tracker-postgres-backup
chmod 0700 /usr/local/sbin/expense-tracker-postgres-backup

cat > /etc/systemd/system/expense-tracker-postgres-backup.service <<'SERVICE'
[Unit]
Description=Expense Tracker PostgreSQL logical backup
After=postgresql.service
Requires=postgresql.service

[Service]
Type=oneshot
ExecStart=/usr/local/sbin/expense-tracker-postgres-backup
SERVICE

cat > /etc/systemd/system/expense-tracker-postgres-backup.timer <<TIMER
[Unit]
Description=Run Expense Tracker PostgreSQL logical backup daily

[Timer]
OnCalendar=*-*-* $BACKUP_TIME $BACKUP_TIMEZONE
Persistent=true
RandomizedDelaySec=15m

[Install]
WantedBy=timers.target
TIMER

systemctl daemon-reload
systemctl enable --now expense-tracker-postgres-backup.timer
if ! systemctl start expense-tracker-postgres-backup.service; then
  journalctl --unit expense-tracker-postgres-backup.service --no-pager --lines 50 >&2 || true
  exit 1
fi
systemctl is-active --quiet expense-tracker-postgres-backup.timer
printf 'Daily PostgreSQL backup timer configured for %s %s and initial backup completed.\n' "$BACKUP_TIME" "$BACKUP_TIMEZONE"
systemd-analyze calendar "*-*-* ${BACKUP_TIME} ${BACKUP_TIMEZONE}"
'''


REMOTE_RESTORE_VERIFICATION = r'''#!/usr/bin/env bash
set -euo pipefail
umask 077

failure_report=""
report_failure() {
  status=$?
  failure_report="$(mktemp /var/tmp/expense-tracker-restore-failure.XXXXXXXX.txt)"
  printf 'failed_at=%s\nfailed_command=%s\nexit_status=%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$BASH_COMMAND" "$status" > "$failure_report"
  if command -v aws >/dev/null; then
    aws s3 cp "$failure_report" "s3://${BACKUP_BUCKET}/verification/restore-failure.txt" --region "$BACKUP_REGION" --sse AES256 --only-show-errors || true
  fi
  exit "$status"
}

cleanup() {
  rm -f "${archive:-}" "${result:-}" "$failure_report" "$0"
}
trap cleanup EXIT
trap report_failure ERR

[[ "$BACKUP_BUCKET" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]] || {
  printf 'invalid backup bucket name\n' >&2
  exit 1
}
[[ "$BACKUP_REGION" =~ ^[a-z]{2}(-[a-z]+)+-[0-9]+$ ]] || {
  printf 'invalid backup region\n' >&2
  exit 1
}

dnf install -y postgresql16 postgresql16-server postgresql16-contrib
postgres_binary="$(rpm -ql postgresql16-server | awk '/\/postgres$/ { value = $0 } END { print value }')"
[[ -x "$postgres_binary" ]] || { printf 'PostgreSQL server binary not found\n' >&2; exit 1; }
"$postgres_binary" --version | grep -Eq '^postgres \(PostgreSQL\) 16\.'
setup_binary="$(command -v postgresql-setup || true)"
[[ -x "$setup_binary" ]] || { printf 'postgresql-setup not found\n' >&2; exit 1; }
if [[ ! -f /var/lib/pgsql/data/PG_VERSION ]]; then
  "$setup_binary" --initdb
fi
[[ "$(</var/lib/pgsql/data/PG_VERSION)" == 16 ]] || { printf 'PGDATA major version is not 16\n' >&2; exit 1; }
systemctl enable --now postgresql.service

key="$(aws s3api list-objects-v2 --bucket "$BACKUP_BUCKET" --prefix daily/ --region "$BACKUP_REGION" --query 'reverse(sort_by(Contents,&LastModified))[0].Key' --output text)"
[[ "$key" == daily/*.dump ]] || { printf 'no logical PostgreSQL backup found\n' >&2; exit 1; }
archive="$(mktemp /var/tmp/expense-tracker-restore.XXXXXXXX.dump)"
aws s3 cp "s3://${BACKUP_BUCKET}/${key}" "$archive" --region "$BACKUP_REGION" --only-show-errors
test -s "$archive"
chown postgres:postgres "$archive"
restore_listing="$(sudo -u postgres pg_restore --list "$archive")"
grep -Eq 'TABLE public (users|groups)' <<< "$restore_listing"
sudo -u postgres createdb expense_tracker_restore_verification
sudo -u postgres pg_restore --exit-on-error --no-owner --no-privileges \
  --dbname=expense_tracker_restore_verification "$archive"

table_count="$(sudo -u postgres psql --dbname expense_tracker_restore_verification --tuples-only --no-align --set ON_ERROR_STOP=1 --command "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users', 'groups', 'expense', 'ledger')")"
[[ "$table_count" == 4 ]] || { printf 'restored schema is incomplete: %s core tables found\n' "$table_count" >&2; exit 1; }
user_count="$(sudo -u postgres psql --dbname expense_tracker_restore_verification --tuples-only --no-align --set ON_ERROR_STOP=1 --command 'SELECT count(*) FROM users')"
group_count="$(sudo -u postgres psql --dbname expense_tracker_restore_verification --tuples-only --no-align --set ON_ERROR_STOP=1 --command 'SELECT count(*) FROM groups')"
verified_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
result="$(mktemp /var/tmp/expense-tracker-restore-result.XXXXXXXX.json)"
printf '{"verified_at":"%s","backup_key":"%s","users":%s,"groups":%s}\n' \
  "$verified_at" "$key" "$user_count" "$group_count" > "$result"
aws s3 cp "$result" "s3://${BACKUP_BUCKET}/verification/latest.json" --region "$BACKUP_REGION" --sse AES256 --only-show-errors
aws s3 rm "s3://${BACKUP_BUCKET}/verification/restore-failure.txt" --region "$BACKUP_REGION" --only-show-errors || true
printf 'Restore verification passed: backup=%s users=%s groups=%s\n' "$key" "$user_count" "$group_count"
'''


REMOTE_BACKUP_DIAGNOSTIC = r'''#!/usr/bin/env bash
set -euo pipefail
umask 077

report="$(mktemp /var/tmp/expense-tracker-backup-diagnostic.XXXXXXXX.txt)"
cleanup() {
  rm -f "$report" "$0"
}
trap cleanup EXIT

{
  printf 'generated_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'aws_cli='; command -v aws || true
  printf 'pg_dump='; command -v pg_dump || true
  systemctl is-enabled expense-tracker-postgres-backup.timer 2>&1 || true
  systemctl is-active expense-tracker-postgres-backup.timer 2>&1 || true
  systemctl status expense-tracker-postgres-backup.service --no-pager --full 2>&1 || true
  journalctl --unit expense-tracker-postgres-backup.service --no-pager --lines 50 2>&1 || true
  if command -v aws >/dev/null; then
    printf 'aws_account='; aws sts get-caller-identity --query Account --output text 2>&1 || true
    aws s3api list-objects-v2 --bucket "$BACKUP_BUCKET" --prefix daily/ --region "$BACKUP_REGION" --max-keys 1 2>&1 || true
  fi
} > "$report"

if ! command -v aws >/dev/null; then
  cat "$report" >&2
  exit 1
fi
aws s3 cp "$report" "s3://${BACKUP_BUCKET}/status/latest.txt" --region "$BACKUP_REGION" --sse AES256 --only-show-errors
cat "$report"
'''


def _ssh_options(key: Path, known_hosts: Path) -> list[str]:
    return [
        "-i", str(key), "-o", "BatchMode=yes", "-o", f"UserKnownHostsFile={known_hosts}",
        "-o", "StrictHostKeyChecking=yes",
    ]


def _wait_for_host(public_ip: str, known_hosts: Path) -> None:
    scanned_output = ""
    for _ in range(30):
        scanned = run(["ssh-keyscan", "-T", "5", public_ip], check=False)
        scanned_output = scanned.stdout
        if scanned.returncode == 0 and scanned_output.strip():
            break
        time.sleep(5)
    if not scanned_output.strip():
        raise CommandError("temporary restore host did not become reachable for SSH")
    known_hosts.write_text(scanned_output)
    os.chmod(known_hosts, 0o600)


def _run_remote(config: Config, public_ip: str, script_content: str, environment: dict[str, str]) -> None:
    key = config.local_credentials.ssh_private_key_file
    suffix = uuid.uuid4().hex
    remote_script = f"/tmp/expense-tracker-postgres-backup-{suffix}.sh"
    with tempfile.TemporaryDirectory(prefix="expense-postgres-backup-") as temporary:
        root = Path(temporary)
        script = root / "script.sh"
        known_hosts = root / "known_hosts"
        script.write_text(script_content)
        os.chmod(script, 0o700)
        _wait_for_host(public_ip, known_hosts)
        options = _ssh_options(key, known_hosts)
        destination = f"ec2-user@{public_ip}"
        run(["scp", *options, str(script), f"{destination}:{remote_script}"], capture=False)
        rendered_environment = " ".join(
            f"{name}={shlex.quote(value)}" for name, value in sorted(environment.items())
        )
        run(
            ["ssh", *options, destination, f"sudo env {rendered_environment} bash {shlex.quote(remote_script)}"],
            capture=False,
        )


def configure(config: Config, outputs: dict[str, Any]) -> None:
    public_ip = str(outputs.get("database_temporary_public_ipv4", ""))
    bucket = str(outputs.get("postgres_backup_bucket_name", ""))
    if not public_ip or not bucket:
        raise CommandError("backup configuration requires temporary database access and a backup bucket")
    _run_remote(
        config,
        public_ip,
        REMOTE_BACKUP_CONFIGURATION,
        {
            "BACKUP_BUCKET": bucket,
            "BACKUP_DATABASE": config.database.name,
            "BACKUP_REGION": config.aws.region,
            "BACKUP_TIME": config.backup.time,
            "BACKUP_TIMEZONE": config.backup.timezone,
        },
    )


def restore_verify(config: Config, outputs: dict[str, Any]) -> None:
    public_ip = str(outputs.get("restore_verification_temporary_public_ipv4", ""))
    bucket = str(outputs.get("postgres_backup_bucket_name", ""))
    if not public_ip or not bucket:
        raise CommandError("restore verification requires a temporary restore host and a backup bucket")
    _run_remote(
        config,
        public_ip,
        REMOTE_RESTORE_VERIFICATION,
        {"BACKUP_BUCKET": bucket, "BACKUP_REGION": config.aws.region},
    )


def diagnose(config: Config, outputs: dict[str, Any]) -> None:
    public_ip = str(outputs.get("database_temporary_public_ipv4", ""))
    bucket = str(outputs.get("postgres_backup_bucket_name", ""))
    if not public_ip or not bucket:
        raise CommandError("backup diagnostics require temporary database access and a backup bucket")
    _run_remote(
        config,
        public_ip,
        REMOTE_BACKUP_DIAGNOSTIC,
        {
            "BACKUP_BUCKET": bucket,
            "BACKUP_DATABASE": config.database.name,
            "BACKUP_REGION": config.aws.region,
        },
    )
