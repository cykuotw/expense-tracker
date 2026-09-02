import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExpenseDateInput } from "./ExpenseDateInput";

describe("ExpenseDateInput", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("formats the visible value and opens the native picker from its calendar action", () => {
        const showPicker = vi.fn();
        Object.defineProperty(HTMLInputElement.prototype, "showPicker", {
            configurable: true,
            value: showPicker,
        });

        render(
            <ExpenseDateInput
                aria-label="Expense date"
                name="occurredOn"
                value="2026-09-02"
                onChange={vi.fn()}
            />
        );

        expect(screen.getByLabelText("Expense date")).toHaveValue("2026/09/02");
        fireEvent.click(screen.getByRole("button", { name: "Choose expense date" }));
        expect(showPicker).toHaveBeenCalledOnce();
    });

    it("keeps the native date value in the form field", () => {
        render(
            <ExpenseDateInput
                aria-label="Expense date"
                name="occurredOn"
                value="2026-09-02"
                onChange={vi.fn()}
            />
        );

        expect(document.querySelector('input[type="date"]')).toHaveValue(
            "2026-09-02"
        );
    });
});
