from __future__ import annotations

import json
import re
import urllib.error
import urllib.request

from common.command import CommandError
from config import Config


def _get(url: str) -> tuple[int, str, dict[str, str]]:
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            headers = {key.lower(): value for key, value in response.headers.items()}
            return response.status, response.read().decode(), headers
    except urllib.error.HTTPError as error:
        headers = {key.lower(): value for key, value in error.headers.items()}
        return error.code, error.read().decode(), headers


FRONTEND_VERSION_PATTERN = re.compile(r"^v-\d{8}-[0-9a-f]{8}$")


def _csp(api_origin: str) -> str:
    return "; ".join(
        (
            "default-src 'self'",
            "base-uri 'self'",
            "object-src 'none'",
            "frame-ancestors 'none'",
            "form-action 'self'",
            "script-src 'self' https://accounts.google.com/gsi/client",
            f"connect-src 'self' {api_origin} https://api.frankfurter.dev https://accounts.google.com/gsi/",
            "frame-src https://accounts.google.com/gsi/",
            "style-src 'self' 'unsafe-inline' https://accounts.google.com/gsi/style",
            "font-src 'self' https://fonts.gstatic.com",
            "img-src 'self' data:",
            "manifest-src 'self'",
            "worker-src 'self'",
        )
    ) + ";"


def _security_headers(config: Config, headers: dict[str, str]) -> None:
    expected = {
        "referrer-policy": "strict-origin-when-cross-origin",
        "x-content-type-options": "nosniff",
        "x-frame-options": "DENY",
        "content-security-policy": _csp(config.api_origin),
    }
    for name, value in expected.items():
        if headers.get(name) != value:
            raise CommandError(
                f"deployed frontend security header does not match: {name}"
            )


def verify(
    config: Config,
    *,
    require_frontend_version: bool = True,
    require_security_headers: bool = True,
) -> None:
    status, document, headers = _get(config.frontend_origin)
    if status != 200 or "runtime-config.js" not in document:
        raise CommandError("deployed frontend document check failed")
    if require_security_headers:
        _security_headers(config, headers)
    status, runtime, _ = _get(f"{config.frontend_origin}/runtime-config.js")
    if status != 200:
        raise CommandError("deployed frontend runtime config is unavailable")
    match = re.search(r"Object\.freeze\((\{.*\})\);", runtime, re.DOTALL)
    if not match:
        raise CommandError("deployed frontend runtime config has an unexpected shape")
    value = json.loads(match.group(1))
    expected = {"apiOrigin": config.api_origin, "apiPath": "/api/v0", "googleOAuthEnabled": True, "googleClientId": config.backend.google_client_id}
    if not isinstance(value, dict) or any(value.get(key) != expected_value for key, expected_value in expected.items()):
        raise CommandError("deployed frontend runtime config does not match deployment config")
    version = value.get("frontendVersion")
    if require_frontend_version and (not isinstance(version, str) or not FRONTEND_VERSION_PATTERN.fullmatch(version)):
        raise CommandError("deployed frontend runtime config does not contain a valid frontend version")
