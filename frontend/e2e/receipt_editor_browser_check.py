"""Real-browser contract check for the development-only receipt editor lab."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from playwright.sync_api import Page, sync_playwright


def prepared_image_contract(page: Page) -> dict[str, object]:
    image = page.get_by_alt_text(
        "Prepared receipt with crop and masks permanently applied"
    )
    return image.evaluate(
        """async (element) => {
            const response = await fetch(element.src);
            const blob = await response.blob();
            const bytes = new Uint8Array(await blob.arrayBuffer());
            const text = new TextDecoder("latin1").decode(bytes);
            const bitmap = await createImageBitmap(blob);
            const canvas = document.createElement("canvas");
            canvas.width = bitmap.width;
            canvas.height = bitmap.height;
            const context = canvas.getContext("2d", { alpha: false });
            context.drawImage(bitmap, 0, 0);
            const pixel = Array.from(context.getImageData(
                Math.floor(bitmap.width * 0.5),
                Math.floor(bitmap.height * 0.5),
                1,
                1,
            ).data);
            bitmap.close();
            canvas.width = 0;
            canvas.height = 0;
            const digest = Array.from(new Uint8Array(
                await crypto.subtle.digest("SHA-256", bytes)
            )).map((value) => value.toString(16).padStart(2, "0")).join("");
            return {
                type: blob.type,
                size: blob.size,
                width: element.naturalWidth,
                height: element.naturalHeight,
                pixel,
                digest,
                metadataMarkers: {
                    exif: text.includes("Exif"),
                    xmp: text.includes("http://ns.adobe.com/xap"),
                    motionPhoto: text.includes("MotionPhoto") || text.includes("MicroVideoOffset"),
                },
            };
        }"""
    )


def run(
    base_url: str, fixture: Path, artifacts_dir: Path | None = None
) -> dict[str, object]:
    source_text = fixture.read_bytes().decode("latin1", errors="ignore")
    source_metadata_markers = {
        "exif": "Exif" in source_text,
        "xmp": "http://ns.adobe.com/xap" in source_text,
        "motionPhoto": "MotionPhoto" in source_text
        or "MicroVideoOffset" in source_text,
    }
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            headless=True,
            executable_path="/usr/bin/google-chrome",
        )
        context = browser.new_context(viewport={"width": 1280, "height": 900})
        page = context.new_page()
        api_requests: list[str] = []
        console_errors: list[str] = []
        page.on(
            "request",
            lambda request: api_requests.append(request.url)
            if request.resource_type in {"fetch", "xhr"}
            and ("/api/" in request.url or "/auth/" in request.url)
            else None,
        )
        page.on(
            "console",
            lambda message: console_errors.append(message.text)
            if message.type == "error"
            else None,
        )

        page.goto(f"{base_url}/__dev/receipt-editor", wait_until="networkidle")
        storage_before = page.evaluate(
            """async () => ({
                local: Object.keys(localStorage),
                session: Object.keys(sessionStorage),
                caches: "caches" in globalThis ? await caches.keys() : [],
                databases: indexedDB.databases ? (await indexedDB.databases()).map((db) => db.name) : [],
            })"""
        )

        page.get_by_label("Choose photo").set_input_files(str(fixture))
        page.get_by_role("heading", name="Crop and protect your receipt").wait_for()
        canvas_box = page.get_by_label("Receipt editing canvas").bounding_box()
        assert canvas_box
        page.mouse.move(
            canvas_box["x"] + canvas_box["width"] * 0.05,
            canvas_box["y"] + canvas_box["height"] * 0.05,
        )
        page.mouse.down()
        page.mouse.move(
            canvas_box["x"] + canvas_box["width"] * 0.95,
            canvas_box["y"] + canvas_box["height"] * 0.95,
            steps=5,
        )
        page.mouse.up()
        page.get_by_role("button", name="Rotate 90°").click()
        page.get_by_role("button", name="Mask area").click()
        canvas_box = page.get_by_label("Receipt editing canvas").bounding_box()
        assert canvas_box
        page.mouse.move(
            canvas_box["x"] + canvas_box["width"] * 0.4,
            canvas_box["y"] + canvas_box["height"] * 0.45,
        )
        page.mouse.down()
        page.mouse.move(
            canvas_box["x"] + canvas_box["width"] * 0.6,
            canvas_box["y"] + canvas_box["height"] * 0.55,
            steps=5,
        )
        page.mouse.up()
        page.get_by_text("1 permanent region", exact=True).wait_for()
        mask_targets = page.evaluate(
            """() => [
                document.querySelector('button[aria-label="Mask 1. Drag to move."]'),
                document.querySelector('button[aria-label="Resize mask 1"]'),
            ].map((element) => {
                const box = element.getBoundingClientRect();
                return { label: element.getAttribute("aria-label"), width: box.width, height: box.height };
            })"""
        )
        if artifacts_dir:
            artifacts_dir.mkdir(parents=True, exist_ok=True)
            page.screenshot(
                path=artifacts_dir / "desktop-editor.jpg",
                full_page=True,
                type="jpeg",
                quality=65,
            )
        page.get_by_role("button", name="Review prepared receipt").click()
        page.get_by_role("heading", name="Review the prepared receipt").wait_for()
        first = prepared_image_contract(page)

        page.get_by_role("button", name="Back to editing").click()
        restored_canvas = page.get_by_label("Receipt editing canvas")
        restored_canvas.wait_for()
        restored_handle = restored_canvas.element_handle()
        assert restored_handle
        page.wait_for_function("(canvas) => canvas.width > 300", arg=restored_handle)
        restored_editor = restored_canvas.evaluate(
            """(canvas) => ({
                width: canvas.width,
                height: canvas.height,
                maskCount: document.querySelectorAll('button[aria-label^="Mask "][aria-label$="Drag to move."]').length,
            })"""
        )
        page.get_by_role("button", name="Zoom in preview").click()
        page.get_by_role("button", name="Zoom in preview").click()
        page.get_by_role("button", name="Review prepared receipt").click()
        page.get_by_role("heading", name="Review the prepared receipt").wait_for()
        after_zoom = prepared_image_contract(page)

        storage_after = page.evaluate(
            """async () => ({
                local: Object.keys(localStorage),
                session: Object.keys(sessionStorage),
                caches: "caches" in globalThis ? await caches.keys() : [],
                databases: indexedDB.databases ? (await indexedDB.databases()).map((db) => db.name) : [],
                horizontalOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
            })"""
        )

        context.close()

        mobile = browser.new_context(viewport={"width": 390, "height": 844})
        mobile_page = mobile.new_page()
        mobile_page.goto(f"{base_url}/__dev/receipt-editor", wait_until="networkidle")
        mobile_page.get_by_label("Choose photo").set_input_files(str(fixture))
        mobile_page.get_by_role("heading", name="Crop and protect your receipt").wait_for()
        mobile_control_names = [
            "Undo",
            "Redo",
            "Reset receipt edits",
            "Crop mode",
            "Mask mode",
            "Pan mode",
            "Rotate receipt 90 degrees",
            "Zoom out preview",
            "Zoom in preview",
            "Fit preview",
        ]
        mobile_controls = []
        for name in mobile_control_names:
            box = mobile_page.get_by_role("button", name=name).bounding_box()
            assert box
            mobile_controls.append(
                {"name": name, "width": box["width"], "height": box["height"]}
            )
        precise_disclosure = mobile_page.locator(
            'button[aria-controls="receipt-precise-controls"]'
        )
        disclosure_box = precise_disclosure.bounding_box()
        assert disclosure_box
        mobile_controls.append(
            {
                "name": "Precise controls",
                "width": disclosure_box["width"],
                "height": disclosure_box["height"],
            }
        )
        assert precise_disclosure.get_attribute("aria-expanded") == "false"
        precise_disclosure.click()
        assert precise_disclosure.get_attribute("aria-expanded") == "true"
        mobile_page.get_by_role("heading", name="Precise crop").wait_for()
        precise_disclosure.click()
        assert precise_disclosure.get_attribute("aria-expanded") == "false"

        mobile_page.get_by_role("button", name="Zoom in preview").click()
        assert mobile_page.get_by_test_id("receipt-viewport-status").text_content() == "125% zoom"
        mobile_page.get_by_role("button", name="Pan mode").click()
        mobile_stage = mobile_page.get_by_label("Receipt editing canvas").locator("xpath=..")
        zoomed_box = mobile_stage.bounding_box()
        assert zoomed_box
        mobile_page.mouse.move(
            zoomed_box["x"] + zoomed_box["width"] / 2,
            zoomed_box["y"] + zoomed_box["height"] / 2,
        )
        mobile_page.mouse.down()
        mobile_page.mouse.move(
            zoomed_box["x"] + zoomed_box["width"] / 2 + 30,
            zoomed_box["y"] + zoomed_box["height"] / 2 + 20,
            steps=4,
        )
        mobile_page.mouse.up()
        mobile_page.get_by_role("button", name="Fit preview").click()
        assert mobile_page.get_by_test_id("receipt-viewport-status").text_content() == "Fit"

        fit_box = mobile_stage.bounding_box()
        viewport_box = mobile_page.get_by_test_id("receipt-stage-viewport").bounding_box()
        canvas_dimensions = mobile_page.get_by_label("Receipt editing canvas").evaluate(
            "(canvas) => ({ width: canvas.width, height: canvas.height })"
        )
        assert fit_box and viewport_box
        stage_is_contained = (
            fit_box["x"] >= viewport_box["x"] - 1
            and fit_box["y"] >= viewport_box["y"] - 1
            and fit_box["x"] + fit_box["width"]
            <= viewport_box["x"] + viewport_box["width"] + 1
            and fit_box["y"] + fit_box["height"]
            <= viewport_box["y"] + viewport_box["height"] + 1
        )
        aspect_error = abs(
            fit_box["width"] / fit_box["height"]
            - canvas_dimensions["width"] / canvas_dimensions["height"]
        )
        if artifacts_dir:
            mobile_page.screenshot(
                path=artifacts_dir / "mobile-editor.jpg",
                full_page=True,
                type="jpeg",
                quality=65,
            )
        mobile_layout = {
            "horizontalOverflow": mobile_page.evaluate(
                "document.documentElement.scrollWidth > document.documentElement.clientWidth"
            ),
            "controls": mobile_controls,
            "fitStatus": mobile_page.get_by_test_id(
                "receipt-viewport-status"
            ).text_content(),
            "stageContained": stage_is_contained,
            "aspectError": aspect_error,
            "stage": fit_box,
            "viewport": viewport_box,
        }
        mobile.close()
        browser.close()

    assert first["type"] == "image/jpeg"
    assert 0 < int(first["size"]) <= 3_670_016
    assert int(first["width"]) <= 4_096 and int(first["height"]) <= 4_096
    assert int(first["width"]) * int(first["height"]) <= 16_000_000
    assert all(channel <= 24 for channel in first["pixel"][:3])
    assert not any(first["metadataMarkers"].values())
    assert first["digest"] == after_zoom["digest"]
    assert int(restored_editor["width"]) > 300
    assert int(restored_editor["height"]) > 150
    assert restored_editor["maskCount"] == 1
    assert storage_before == {key: storage_after[key] for key in storage_before}
    assert not storage_after["horizontalOverflow"]
    assert not mobile_layout["horizontalOverflow"]
    assert all(
        control["width"] >= 44 and control["height"] >= 44
        for control in mobile_layout["controls"]
    )
    assert mobile_layout["fitStatus"] == "Fit"
    assert mobile_layout["stageContained"]
    assert mobile_layout["aspectError"] < 0.01
    assert all(
        target["width"] >= 44 and target["height"] >= 44
        for target in mask_targets
    )
    assert not api_requests, api_requests
    assert not console_errors, console_errors

    return {
        "fixture": fixture.name,
        "sourceMetadataMarkers": source_metadata_markers,
        "output": {
            "type": first["type"],
            "bytes": first["size"],
            "width": first["width"],
            "height": first["height"],
            "maskPixel": first["pixel"],
            "metadataMarkers": first["metadataMarkers"],
        },
        "zoomPanIndependent": first["digest"] == after_zoom["digest"],
        "editorCanvasRestored": restored_editor,
        "browserPersistenceUnchanged": storage_before
        == {key: storage_after[key] for key in storage_before},
        "apiRequests": api_requests,
        "desktopHorizontalOverflow": storage_after["horizontalOverflow"],
        "maskPointerTargets": mask_targets,
        "mobile": mobile_layout,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://127.0.0.1:4175")
    parser.add_argument("--fixture", type=Path, required=True)
    parser.add_argument("--artifacts-dir", type=Path)
    args = parser.parse_args()
    if not args.fixture.is_file():
        raise SystemExit(f"fixture does not exist: {args.fixture}")
    print(
        json.dumps(
            run(args.base_url, args.fixture.resolve(), args.artifacts_dir), indent=2
        )
    )


if __name__ == "__main__":
    main()
