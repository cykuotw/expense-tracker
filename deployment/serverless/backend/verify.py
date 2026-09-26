from __future__ import annotations

import http.cookiejar
import json
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Mapping

from common.command import CommandError
from config import Config


@dataclass(frozen=True)
class Response:
    status: int
    headers: Mapping[str, str]
    body: bytes


PREFLIGHT_PROPAGATION_TIMEOUT_SECONDS = 60
PREFLIGHT_RETRY_INTERVAL_SECONDS = 3


def _request(url: str, *, method: str = "GET", headers: dict[str, str] | None = None, body: object | None = None, opener: urllib.request.OpenerDirector | None = None) -> Response:
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(url, data=data, method=method, headers=headers or {})
    try:
        response = (opener or urllib.request.build_opener()).open(request, timeout=20)
        return Response(response.status, response.headers, response.read())
    except urllib.error.HTTPError as error:
        return Response(error.code, error.headers, error.read())


def _cors_values(response: Response, name: str) -> set[str]:
    return {
        value.strip().lower()
        for value in response.headers.get(name, "").split(",")
        if value.strip()
    }


def _wait_for_cors_preflight(
    url: str,
    *,
    origin: str,
    method: str,
    request_headers: tuple[str, ...],
    required_headers: tuple[str, ...] = (),
    failure_message: str,
) -> None:
    headers = {
        "Origin": origin,
        "Access-Control-Request-Method": method,
        "Access-Control-Request-Headers": ", ".join(request_headers),
    }

    def ready() -> bool:
        response = _request(url, method="OPTIONS", headers=headers)
        allowed_methods = _cors_values(response, "Access-Control-Allow-Methods")
        allowed_headers = _cors_values(response, "Access-Control-Allow-Headers")
        return (
            response.status == 204
            and response.headers.get("Access-Control-Allow-Origin") == origin
            and method.lower() in allowed_methods
            and all(header.lower() in allowed_headers for header in required_headers)
        )

    if ready():
        return
    deadline = time.monotonic() + PREFLIGHT_PROPAGATION_TIMEOUT_SECONDS
    while time.monotonic() < deadline:
        time.sleep(PREFLIGHT_RETRY_INTERVAL_SECONDS)
        if ready():
            return
    raise CommandError(failure_message)


def verify_api(
    config: Config,
    raw_endpoint: str | None = None,
    *,
    raw_disabled: bool = False,
    require_patch_cors: bool = True,
    require_ocr_cors: bool = True,
    require_google_register_authorizer: bool = True,
    require_google_link_authorizer: bool = True,
) -> None:
    api = f"{config.api_origin}/api/v0"
    health = _request(f"{api}/health")
    if health.status != 200:
        raise CommandError(f"custom API health returned HTTP {health.status}")

    origin = config.frontend_origin
    preflight = _request(f"{api}/auth/google/exchange", method="OPTIONS", headers={
        "Origin": origin,
        "Access-Control-Request-Method": "POST",
        "Access-Control-Request-Headers": "Authorization, X-CSRF-Token",
    })
    if preflight.status != 204 or preflight.headers.get("Access-Control-Allow-Origin") != origin:
        raise CommandError("allowed credentialed CORS preflight failed")
    if require_patch_cors:
        _wait_for_cors_preflight(
            f"{api}/admin/users/00000000-0000-0000-0000-000000000000/role",
            origin=origin,
            method="PATCH",
            request_headers=("Content-Type", "Authorization", "X-CSRF-Token"),
            failure_message="PATCH CORS preflight failed after propagation timeout",
        )
    if require_ocr_cors:
        _wait_for_cors_preflight(
            f"{api}/ocr/drafts",
            origin=origin,
            method="POST",
            request_headers=("Content-Type", "Authorization", "X-OCR-Account-ID", "X-OCR-Request-ID"),
            required_headers=("Content-Type", "Authorization", "X-OCR-Account-ID", "X-OCR-Request-ID"),
            failure_message="OCR CORS preflight failed after propagation timeout",
        )
    disallowed = _request(f"{api}/auth/csrf", headers={"Origin": "https://invalid.example"})
    if disallowed.headers.get("Access-Control-Allow-Origin"):
        raise CommandError("disallowed CORS origin was reflected")
    google_routes = [("auth/google/exchange", "exchange")]
    if require_google_register_authorizer:
        google_routes.append(("auth/google/register", "register"))
    if require_google_link_authorizer:
        google_routes.append(("account/google/link", "link"))
    for path, label in google_routes:
        for authorization in (None, "Bearer not-a-jwt"):
            headers = {} if authorization is None else {"Authorization": authorization}
            response = _request(f"{api}/{path}", method="POST", headers=headers)
            if response.status != 401:
                raise CommandError(
                    f"Google {label} authorizer negative check returned HTTP {response.status}"
                )
    if raw_endpoint:
        raw_url = f"{raw_endpoint.rstrip('/')}/api/v0/health"
        if raw_disabled:
            deadline = time.monotonic() + 120
            while _request(raw_url).status < 400:
                if time.monotonic() >= deadline:
                    raise CommandError("raw execute-api endpoint is still reachable after 120 seconds")
                time.sleep(3)
        elif _request(raw_url).status != 200:
            raise CommandError("raw execute-api endpoint failed before cutover")


def verify_session(config: Config) -> None:
    if not config.first_admin:
        return
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    root = f"{config.api_origin}/api/v0"
    common = {"Origin": config.frontend_origin, "Content-Type": "application/json"}
    csrf_response = _request(f"{root}/auth/csrf", headers={"Origin": config.frontend_origin}, opener=opener)
    if csrf_response.status != 200:
        raise CommandError("CSRF token request failed")
    token = json.loads(csrf_response.body).get("csrfToken")
    if not token:
        raise CommandError("CSRF response did not contain csrfToken")
    headers = {**common, "X-CSRF-Token": token}
    login = _request(f"{root}/login", method="POST", headers=headers, body={"email": config.first_admin.email, "password": config.first_admin.password}, opener=opener)
    if login.status != 200:
        raise CommandError(f"deployed local login returned HTTP {login.status}")
    if _request(f"{root}/auth/me", headers={"Origin": config.frontend_origin}, opener=opener).status != 200:
        raise CommandError("auth/me failed after login")
    if _request(f"{root}/auth/refresh", method="POST", headers=headers, opener=opener).status != 200:
        raise CommandError("session refresh failed")
    if _request(f"{root}/logout", method="POST", headers=headers, opener=opener).status not in {200, 204}:
        raise CommandError("logout failed")
    if _request(f"{root}/auth/me", headers={"Origin": config.frontend_origin}, opener=opener).status not in {401, 403}:
        raise CommandError("session remained authenticated after logout")


def verify_google_exchange(config: Config) -> None:
    token_file = config.local_credentials.google_id_token_file
    if token_file is None:
        return
    token = token_file.read_text().strip()
    if not token:
        raise CommandError("Google ID token file is empty")
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    root = f"{config.api_origin}/api/v0"
    csrf_response = _request(f"{root}/auth/csrf", headers={"Origin": config.frontend_origin}, opener=opener)
    if csrf_response.status != 200:
        raise CommandError("CSRF token request failed before Google exchange")
    csrf = json.loads(csrf_response.body).get("csrfToken")
    headers = {
        "Origin": config.frontend_origin,
        "Authorization": f"Bearer {token}",
        "X-CSRF-Token": csrf or "",
    }
    exchange = _request(f"{root}/auth/google/exchange", method="POST", headers=headers, opener=opener)
    if exchange.status != 200:
        raise CommandError(f"valid Google exchange returned HTTP {exchange.status}")
    if _request(f"{root}/auth/me", headers={"Origin": config.frontend_origin}, opener=opener).status != 200:
        raise CommandError("Google exchange did not create an application session")
