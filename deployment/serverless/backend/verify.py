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


def _request(url: str, *, method: str = "GET", headers: dict[str, str] | None = None, body: object | None = None, opener: urllib.request.OpenerDirector | None = None) -> Response:
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(url, data=data, method=method, headers=headers or {})
    try:
        response = (opener or urllib.request.build_opener()).open(request, timeout=20)
        return Response(response.status, response.headers, response.read())
    except urllib.error.HTTPError as error:
        return Response(error.code, error.headers, error.read())


def verify_api(
    config: Config,
    raw_endpoint: str | None = None,
    *,
    raw_disabled: bool = False,
    require_patch_cors: bool = True,
    require_google_register_authorizer: bool = True,
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
        patch_preflight = _request(f"{api}/admin/users/00000000-0000-0000-0000-000000000000/role", method="OPTIONS", headers={
            "Origin": origin,
            "Access-Control-Request-Method": "PATCH",
            "Access-Control-Request-Headers": "Content-Type, Authorization, X-CSRF-Token",
        })
        allowed_methods = patch_preflight.headers.get("Access-Control-Allow-Methods", "")
        if patch_preflight.status != 204 or "PATCH" not in {method.strip() for method in allowed_methods.split(",")}:
            raise CommandError("PATCH CORS preflight failed")
    disallowed = _request(f"{api}/auth/csrf", headers={"Origin": "https://invalid.example"})
    if disallowed.headers.get("Access-Control-Allow-Origin"):
        raise CommandError("disallowed CORS origin was reflected")
    google_paths = ["exchange"]
    if require_google_register_authorizer:
        google_paths.append("register")
    for google_path in google_paths:
        for authorization in (None, "Bearer not-a-jwt"):
            headers = {} if authorization is None else {"Authorization": authorization}
            response = _request(f"{api}/auth/google/{google_path}", method="POST", headers=headers)
            if response.status != 401:
                raise CommandError(
                    f"Google {google_path} authorizer negative check returned HTTP {response.status}"
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
