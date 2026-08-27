from __future__ import annotations

import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock


ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from backend.verify import Response, verify_api
from common.command import CommandError


class VerifyTest(unittest.TestCase):
    def test_raw_endpoint_disable_waits_for_propagation(self) -> None:
        config = SimpleNamespace(
            api_origin="https://api.example.com",
            frontend_origin="https://expense.example.com",
        )
        responses = [
            Response(200, {}, b""),
            Response(204, {"Access-Control-Allow-Origin": config.frontend_origin}, b""),
            Response(204, {"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, OPTIONS"}, b""),
            Response(200, {}, b""),
            Response(401, {}, b""),
            Response(401, {}, b""),
            Response(401, {}, b""),
            Response(401, {}, b""),
            Response(200, {}, b""),
            Response(404, {}, b""),
        ]
        with mock.patch("backend.verify._request", side_effect=responses), \
             mock.patch("backend.verify.time.monotonic", side_effect=[10, 11]), \
             mock.patch("backend.verify.time.sleep") as sleep:
            verify_api(config, "https://raw.example.com", raw_disabled=True)
        sleep.assert_called_once_with(3)

    def test_baseline_verification_does_not_require_patch_cors(self) -> None:
        config = SimpleNamespace(
            api_origin="https://api.example.com",
            frontend_origin="https://expense.example.com",
        )
        responses = [
            Response(200, {}, b""),
            Response(204, {"Access-Control-Allow-Origin": config.frontend_origin}, b""),
            Response(200, {}, b""),
            Response(401, {}, b""),
            Response(401, {}, b""),
        ]
        with mock.patch("backend.verify._request", side_effect=responses) as request:
            verify_api(
                config,
                require_patch_cors=False,
                require_google_register_authorizer=False,
            )
        self.assertEqual(request.call_count, 5)

    def test_google_register_authorizer_is_required_after_update(self) -> None:
        config = SimpleNamespace(
            api_origin="https://api.example.com",
            frontend_origin="https://expense.example.com",
        )
        responses = [
            Response(200, {}, b""),
            Response(204, {"Access-Control-Allow-Origin": config.frontend_origin}, b""),
            Response(204, {"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, OPTIONS"}, b""),
            Response(200, {}, b""),
            Response(401, {}, b""),
            Response(401, {}, b""),
            Response(404, {}, b""),
        ]
        with mock.patch("backend.verify._request", side_effect=responses):
            with self.assertRaisesRegex(
                CommandError,
                "Google register authorizer negative check returned HTTP 404",
            ):
                verify_api(config)


if __name__ == "__main__":
    unittest.main()
