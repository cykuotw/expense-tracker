import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import AmountCalculator from "./AmountCalculator";

beforeEach(() => {
    Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.setAttribute("open", ""); } });
    Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.removeAttribute("open"); } });
});
afterEach(() => {
    cleanup();
    Reflect.deleteProperty(HTMLDialogElement.prototype, "showModal");
    Reflect.deleteProperty(HTMLDialogElement.prototype, "close");
    vi.restoreAllMocks();
});

describe("AmountCalculator", () => {
    it("previews amounts and applies only on explicit confirmation without submitting the expense", () => {
        const onApply = vi.fn();
        const submit = vi.fn((event) => event.preventDefault());
        render(<form onSubmit={submit}><AmountCalculator amount="12.50" amountDigits={2} currency="CAD" onApply={onApply} /></form>);
        fireEvent.click(screen.getByRole("button", { name: "Calculate amount" }));
        expect(screen.getByLabelText("Calculation")).toHaveValue("12.50");
        expect(document.activeElement).toBe(screen.getByRole("button", { name: "Cancel" }));
        fireEvent.click(screen.getByRole("button", { name: "Add" }));
        fireEvent.click(screen.getByRole("button", { name: "8" }));
        expect(screen.getByText("20.50 CAD")).toBeVisible();
        expect(onApply).not.toHaveBeenCalled();
        fireEvent.keyDown(screen.getByLabelText("Calculation"), { key: "Enter" });
        expect(onApply).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "Apply amount" }));
        expect(onApply).toHaveBeenCalledExactlyOnceWith("20.50");
        expect(submit).not.toHaveBeenCalled();
        expect(screen.queryByRole("dialog")).toBeNull();
        expect(document.activeElement).toBe(screen.getByRole("button", { name: "Calculate amount" }));
    });
    it("preserves the original amount on cancel and Escape", () => {
        const onApply = vi.fn();
        render(<AmountCalculator amount="10.00" amountDigits={2} currency="CAD" onApply={onApply} />);
        const trigger = screen.getByRole("button", { name: "Calculate amount" });
        fireEvent.click(trigger);
        fireEvent.change(screen.getByLabelText("Calculation"), { target: { value: "1+2" } });
        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
        expect(onApply).not.toHaveBeenCalled();
        fireEvent.click(trigger);
        expect(screen.getByLabelText("Calculation")).toHaveValue("10.00");
        fireEvent(screen.getByRole("dialog"), new Event("cancel", { cancelable: true }));
        expect(screen.queryByRole("dialog")).toBeNull();
        expect(onApply).not.toHaveBeenCalled();
    });
    it("explains rounding and prevents applying an invalid result", () => {
        render(<AmountCalculator amount="" amountDigits={0} currency="JPY" onApply={vi.fn()} />);
        fireEvent.click(screen.getByRole("button", { name: "Calculate amount" }));
        fireEvent.change(screen.getByLabelText("Calculation"), { target: { value: "10/3" } });
        expect(screen.getByText("3 JPY")).toBeVisible();
        expect(screen.getByText(/Rounded to 0 decimal places/)).toBeVisible();
        fireEvent.change(screen.getByLabelText("Calculation"), { target: { value: "1/0" } });
        expect(screen.getByText("Cannot divide by zero.")).toBeVisible();
        expect(screen.getByRole("button", { name: "Apply amount" })).toBeDisabled();
    });
    it("disables the calculator without currency precision", () => {
        render(<AmountCalculator amount="" amountDigits={null} currency="" onApply={vi.fn()} />);
        expect(screen.getByRole("button", { name: "Calculate amount" })).toBeDisabled();
    });
    it("keeps Tab and Shift+Tab within the calculator", () => {
        render(<AmountCalculator amount="1.00" amountDigits={2} currency="CAD" onApply={vi.fn()} />);
        fireEvent.click(screen.getByRole("button", { name: "Calculate amount" }));
        const cancel = screen.getByRole("button", { name: "Cancel" });
        const apply = screen.getByRole("button", { name: "Apply amount" });
        fireEvent.keyDown(cancel, { key: "Tab", shiftKey: true });
        expect(apply).toHaveFocus();
        fireEvent.keyDown(apply, { key: "Tab" });
        expect(cancel).toHaveFocus();
    });
});
