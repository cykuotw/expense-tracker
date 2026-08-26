from __future__ import annotations

import json
import re
import urllib.error
import urllib.request

from common.command import CommandError
from config import Config


def _get(url: str) -> tuple[int, str]:
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            return response.status, response.read().decode()
    except urllib.error.HTTPError as error:
        return error.code, error.read().decode()


def verify(config: Config) -> None:
    status, document = _get(config.frontend_origin)
    if status != 200 or "runtime-config.js" not in document:
        raise CommandError("deployed frontend document check failed")
    status, runtime = _get(f"{config.frontend_origin}/runtime-config.js")
    if status != 200:
        raise CommandError("deployed frontend runtime config is unavailable")
    match = re.search(r"Object\.freeze\((\{.*\})\);", runtime, re.DOTALL)
    if not match:
        raise CommandError("deployed frontend runtime config has an unexpected shape")
    value = json.loads(match.group(1))
    expected = {"apiOrigin": config.api_origin, "apiPath": "/api/v0", "googleOAuthEnabled": True, "googleClientId": config.backend.google_client_id}
    if value != expected:
        raise CommandError("deployed frontend runtime config does not match deployment config")
