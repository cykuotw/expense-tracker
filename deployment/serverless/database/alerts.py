"""Install or stop the DB connection timer through the guarded host setup path."""
from __future__ import annotations

import base64
import re
from pathlib import Path
from typing import Any

from common.command import CommandError
from config import Config
from database import backup


def installation_script(bucket: str, region: str) -> str:
    if not re.fullmatch(r"[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]", bucket) or not re.fullmatch(r"[a-z]{2}(-[a-z]+)+-[0-9]+", region):
        raise CommandError("invalid database alert relay configuration")
    program = base64.b64encode(Path(__file__).with_name("connection_monitor.py").read_bytes()).decode()
    return f'''#!/usr/bin/env bash
set -euo pipefail
program=""
trap 'rm -f "$0" "$program"' EXIT
command -v aws >/dev/null
command -v psql >/dev/null
command -v python3 >/dev/null
install -d -o root -g root -m 0700 /var/lib/expense-tracker-db-alerts
install -d -o root -g root -m 0755 /usr/local/libexec
program=$(mktemp /usr/local/libexec/.db-alerts.XXXXXX)
printf '%s' '{program}' | base64 --decode > "$program"
chown root:root "$program"
chmod 0700 "$program"
mv -f "$program" /usr/local/libexec/expense-tracker-db-alerts.py
cat > /etc/systemd/system/expense-tracker-db-alerts.service <<'SERVICE'
[Unit]
Description=Check PostgreSQL instance connection pressure
After=postgresql.service
[Service]
Type=oneshot
UMask=0077
TimeoutStartSec=45
ExecStart=/usr/bin/python3 /usr/local/libexec/expense-tracker-db-alerts.py --bucket {bucket} --region {region}
SERVICE
cat > /etc/systemd/system/expense-tracker-db-alerts.timer <<'TIMER'
[Unit]
Description=Check PostgreSQL connections every minute
[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
AccuracySec=1s
[Install]
WantedBy=timers.target
TIMER
systemctl daemon-reload
systemctl enable --now expense-tracker-db-alerts.timer
'''


DISABLE_SCRIPT = '''#!/usr/bin/env bash
set -euo pipefail
trap 'rm -f "$0"' EXIT
if systemctl cat expense-tracker-db-alerts.timer >/dev/null 2>&1; then
    systemctl disable --now expense-tracker-db-alerts.timer
    systemctl stop expense-tracker-db-alerts.service
fi
'''


def configure(config: Config, outputs: dict[str, Any]) -> None:
    public_ip = str(outputs.get("database_temporary_public_ipv4", ""))
    if not public_ip:
        raise CommandError("database monitor configuration requires guarded temporary host access")
    script = DISABLE_SCRIPT
    if config.database_alerting_enabled:
        bucket = str(outputs.get("postgres_backup_bucket_name", ""))
        script = installation_script(bucket, config.aws.region)
    # Only a public bucket name and region enter the host script. The webhook
    # remains exclusively in the outside-VPC notifier's protected environment.
    backup._run_remote(config, public_ip, script, {})
