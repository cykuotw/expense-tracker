import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useFormFocusVisibility } from "./useFormFocusVisibility";

function Form({ ready = true }: { ready?: boolean }) {
    const ref = useFormFocusVisibility();
    if (!ready) return <p>Loading form</p>;
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

    it("attaches after loading and cancels pending work when the form disappears", () => {
        const { rerender } = render(<Form ready={false} />);
        rerender(<Form />);
        const amount = screen.getByLabelText("Amount");
        const page = amount.closest("[data-form-field]")!.parentElement!;
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
        act(() => amount.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 32, behavior: "smooth" });
        vi.mocked(window.scrollBy).mockClear();
        fireEvent.resize(window);
        rerender(<Form ready={false} />);
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
        expect(page.style.getPropertyValue("--form-keyboard-space")).toBe("");
        rerender(<Form />);
        const description = screen.getByLabelText("Description");
        vi.spyOn(description, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
        act(() => description.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 32, behavior: "smooth" });
    });

    it("reveals only the latest focus and resumes after touch scrolling on a new focus", () => {
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        const description = screen.getByLabelText("Description");
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 700, bottom: 760, height: 60 } as DOMRect);
        vi.spyOn(description, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
        act(() => amount.focus());
        fireEvent.touchMove(window);
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
        act(() => { amount.blur(); amount.focus(); description.focus(); });
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 32, behavior: "smooth" });
    });

    it("includes a field's caption and described feedback, prioritizing the input when feedback is oversized", () => {
        render(<Form />);
        const amount = screen.getByLabelText("Amount");
        const field = amount.closest("[data-form-field]")!;
        const caption = field.querySelector("label")!;
        const feedback = document.createElement("p");
        feedback.id = "amount-feedback";
        field.append(feedback);
        amount.setAttribute("aria-describedby", feedback.id);
        vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 530, bottom: 570, height: 40 } as DOMRect);
        vi.spyOn(caption, "getBoundingClientRect").mockReturnValue({ top: 500, bottom: 520, height: 20 } as DOMRect);
        vi.spyOn(feedback, "getBoundingClientRect").mockReturnValue({ top: 580, bottom: 620, height: 40 } as DOMRect);
        act(() => amount.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 32, behavior: "smooth" });
        vi.mocked(window.scrollBy).mockClear();
        vi.spyOn(feedback, "getBoundingClientRect").mockReturnValue({ top: 580, bottom: 1200, height: 620 } as DOMRect);
        fireEvent.resize(window);
        settle();
        expect(window.scrollBy).not.toHaveBeenCalled();
    });

    it("includes the caption when a wrapping label is the field boundary", () => {
        function WrappedField() {
            const ref = useFormFocusVisibility();
            return <div ref={ref}><label data-form-field>First name<input /></label></div>;
        }
        render(<WrappedField />);
        const input = screen.getByRole("textbox", { name: "First name" });
        vi.spyOn(input, "getBoundingClientRect").mockReturnValue({ top: 20, bottom: 60, height: 40 } as DOMRect);
        vi.spyOn(input.closest("label")!, "getBoundingClientRect").mockReturnValue({ top: -8, bottom: 60, height: 68 } as DOMRect);
        act(() => input.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: -20, behavior: "smooth" });
    });

    it("keeps a checkbox's complete member label visible", () => {
        function MemberField() {
            const ref = useFormFocusVisibility();
            return <div ref={ref}><label data-form-field><input type="checkbox" />Friend name and email</label></div>;
        }
        render(<MemberField />);
        const checkbox = screen.getByRole("checkbox", { name: "Friend name and email" });
        vi.spyOn(checkbox, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 580, height: 20 } as DOMRect);
        vi.spyOn(checkbox.closest("label")!, "getBoundingClientRect").mockReturnValue({ top: 544, bottom: 600, height: 56 } as DOMRect);
        act(() => checkbox.focus());
        settle();
        expect(window.scrollBy).toHaveBeenCalledExactlyOnceWith({ top: 12, behavior: "smooth" });
    });

    it.each(['data-picker-panel', 'role="dialog"', 'role="alertdialog"'])(
        "does not move the background when focus is inside %s", (attribute) => {
            render(<Form />);
            const amount = screen.getByLabelText("Amount");
            const panel = document.createElement("div");
            if (attribute === "data-picker-panel") panel.setAttribute(attribute, "");
            else panel.setAttribute("role", attribute.includes("alertdialog") ? "alertdialog" : "dialog");
            amount.parentElement!.append(panel);
            panel.append(amount);
            vi.spyOn(amount, "getBoundingClientRect").mockReturnValue({ top: 560, bottom: 620, height: 60 } as DOMRect);
            act(() => amount.focus());
            settle();
            expect(window.scrollBy).not.toHaveBeenCalled();
        },
    );
});
