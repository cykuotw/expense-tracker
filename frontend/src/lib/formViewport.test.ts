import { afterEach, describe, expect, it, vi } from "vitest";
import { getFormViewport, pickerPlacement, revealDelta } from "./formViewport";

afterEach(() => { document.body.innerHTML = ""; vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe("form viewport", () => {
    const bounds = { top: 80, bottom: 540, left: 12, right: 363 };

    it("opens upward near navigation and fits the remaining space", () => {
        const placement = pickerPlacement({ top: 420, bottom: 476, left: 16, width: 343 }, bounds, 320);
        expect(placement).toEqual({ side: "above", top: 92, left: 16, width: 343, height: 320 });
    });
    it("constrains a tall picker to a short viewport without horizontal overflow", () => {
        const placement = pickerPlacement({ top: 200, bottom: 256, left: 0, width: 400 }, bounds, 500);
        expect(placement.side).toBe("below");
        expect(placement.top + placement.height).toBe(bounds.bottom);
        expect(placement.left).toBe(bounds.left);
        expect(placement.width).toBe(bounds.right - bounds.left);
    });
    it("scrolls only as much as needed and does not center a visible field", () => {
        expect(revealDelta({ top: 100, bottom: 180 }, bounds)).toBe(0);
        expect(revealDelta({ top: 510, bottom: 570 }, bounds)).toBe(30);
        expect(revealDelta({ top: 60, bottom: 120 }, bounds)).toBe(-20);
    });
    it("uses the visible area when the keyboard leaves neither side with enough room", () => {
        const placement = pickerPlacement({ top: 160, bottom: 216, left: 16, width: 343 }, { ...bounds, bottom: 300 }, 384);
        expect(placement.height).toBe(220);
        expect(placement.top).toBe(80);
        expect(placement.top + placement.height).toBe(300);
    });
    it("uses the visible keyboard viewport and actual safe-area navigation height", () => {
        const viewport = { offsetTop: 20, offsetLeft: 0, height: 400, width: 375 };
        vi.stubGlobal("visualViewport", viewport);
        document.body.innerHTML = '<nav class="app-shell__mobile-nav"></nav><header class="mobile-page-header--sticky"></header>';
        vi.spyOn(document.querySelector("nav")!, "getBoundingClientRect").mockReturnValue({ top: 370, bottom: 420, height: 50 } as DOMRect);
        vi.spyOn(document.querySelector("header")!, "getBoundingClientRect").mockReturnValue({ top: 20, bottom: 76, height: 56 } as DOMRect);
        expect(getFormViewport()).toEqual({ top: 88, bottom: 358, left: 12, right: 363 });
    });
});
