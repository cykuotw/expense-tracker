import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useFormFocusVisibility } from "./useFormFocusVisibility";

function Form() {
    const ref = useFormFocusVisibility();
    return <div ref={ref}><div data-form-field><label htmlFor="amount">Amount</label><label className="expense-form-input-shell"><input id="amount" required /></label></div><input aria-label="Description" required /></div>;
}

describe("form focus visibility", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.stubGlobal("scrollBy", vi.fn());
        vi.stubGlobal("innerHeight", 600);
        vi.stubGlobal("matchMedia", vi.fn(() => ({ matches: false })));
    });
    afterEach(() => { cleanup(); vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
    const settle = () => act(() => { vi.advanceTimersByTime(200); });

    it("waits for native adjustments and moves only an obscured field", () => {
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
        act(() => amount.focus());
        expect(window.scrollBy).not.toHaveBeenCalled();
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 32, behavior: "smooth" });
    });
    it("does not pull back after manual scrolling or recenter a visible field", () => {
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 100, bottom: 160, height: 60 } as DOMRect);
        act(() => amount.focus());
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
        fireEvent.wheel(window);
        fireEvent.resize(window);
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
    });
    it("keeps the input shell and focus ring visible, not only the inner text", () => {
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 566, bottom: 590, height: 24 } as DOMRect);
        vi.spyOn(amount.closest(".expense-form-input-shell")!, "getBoundingClientRect").mockReturnValue({ top: 550, bottom: 606, height: 56 } as DOMRect);
        act(() => amount.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 18, behavior: "smooth" });
    });
    it("responds to keyboard resize and respects reduced motion", () => {
        const viewport = new EventTarget();
        Object.assign(viewport, { height: 600, width: 375, offsetTop: 0, offsetLeft: 0, scale: 1 });
        vi.stubGlobal("visualViewport", viewport);
        vi.stubGlobal("matchMedia", vi.fn(() => ({ matches: true })));
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 350, bottom: 410, height: 60 } as DOMRect);
        act(() => amount.focus());
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
        Object.assign(viewport, { height: 400 });
        act(() => viewport.dispatchEvent(new Event("resize")));
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 22, behavior: "instant" });
    });
    it("reveals the first invalid field without changing validation", () => {
        render(<Form />);
        fireEvent.invalid(screen.getByLabelText("Amount"));
        fireEvent.invalid(screen.getByLabelText("Description"));
        settle();
        expect(document.activeElement).toBe(screen.getByLabelText("Amount"));
        expect((screen.getByLabelText("Amount") as HTMLInputElement).validity.valueMissing).toBe(true);
    });
});
