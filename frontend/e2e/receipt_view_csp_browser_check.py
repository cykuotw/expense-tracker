"""Check synthetic receipt images through the built worker under real browser CSP.

Run with uv after building frontend/dist. No production images or credentials
are used; local origins stand in for the application and its private S3 host.
"""

from __future__ import annotations

import argparse
import mimetypes
import sys
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Thread
from urllib.parse import urlsplit

from playwright.sync_api import Browser, sync_playwright

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "deployment/serverless"))
from frontend.verify import _csp


@contextmanager
def serve(handler: type[BaseHTTPRequestHandler]):
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


class ImageHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        image = b'<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"><rect width="1" height="1"/></svg>'
        self.send_response(200)
        self.send_header("Content-Type", "image/svg+xml")
        self.send_header("Cache-Control", "private, no-store")
        self.end_headers()
        self.wfile.write(image)

    def log_message(self, *_):
        pass


def check_policy(browser: Browser, dist: Path, image_origin: str, allow_connect: bool):
    policy = _csp("https://api.example.test", image_origin)
    if not allow_connect:
        # Match production's original policy: img-src allowed the image,
        # while fetch() from the worker was blocked by connect-src.
        policy = policy.replace(f" {image_origin}", "", 1)

    class ApplicationHandler(BaseHTTPRequestHandler):
        def do_GET(self):
            path = urlsplit(self.path).path
            if path == "/receipt-check.html":
                body = b'<!doctype html><title>Receipt CSP check</title><img id="receipt" alt="Synthetic receipt">'
                content_type = "text/html"
            else:
                file = (dist / path.lstrip("/")).resolve()
                if not file.is_relative_to(dist) or not file.is_file():
                    self.send_error(404)
                    return
                body = file.read_bytes()
                content_type = mimetypes.guess_type(file.name)[0] or "application/octet-stream"
            self.send_response(200)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Security-Policy", policy)
            self.send_header("Cache-Control", "no-cache")
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_):
            pass

    with serve(ApplicationHandler) as origin:
        context = browser.new_context(service_workers="allow")
        try:
            page = context.new_page()
            page.goto(f"{origin}/receipt-check.html")
            page.wait_for_load_state("networkidle")
            page.evaluate("async () => { await navigator.serviceWorker.register('/service-worker.js'); await navigator.serviceWorker.ready; }")
            page.wait_for_function("navigator.serviceWorker.controller !== null")
            loaded = page.evaluate("""url => new Promise(resolve => {
                const image = document.getElementById('receipt');
                image.onload = () => resolve(image.naturalWidth > 0);
                image.onerror = () => resolve(false);
                image.src = url;
            })""", f"{image_origin}/retained/synthetic.svg?X-Amz-Signature=synthetic")
            assert loaded == allow_connect, f"Unexpected image result for connect-src={allow_connect}"
            cached = page.evaluate("""async () => {
                const requests = [];
                for (const name of await caches.keys()) {
                    requests.push(...await (await caches.open(name)).keys());
                }
                return requests.map(request => request.url);
            }""")
            assert not any("X-Amz-Signature" in url for url in cached), "Signed receipt was cached"
            print(f"connect-src receipt origin={allow_connect}: image_loaded={loaded}, signed_urls_cached=0")
        finally:
            context.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dist", type=Path, default=ROOT / "frontend/dist")
    args = parser.parse_args()
    dist = args.dist.resolve()
    if not (dist / "service-worker.js").is_file():
        parser.error("Build the frontend before running this check")
    with serve(ImageHandler) as image_origin, sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        try:
            check_policy(browser, dist, image_origin, allow_connect=False)
            check_policy(browser, dist, image_origin, allow_connect=True)
        finally:
            browser.close()


if __name__ == "__main__":
    main()
