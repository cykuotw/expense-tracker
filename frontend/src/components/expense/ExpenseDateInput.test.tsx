import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExpenseDateInput } from "./ExpenseDateInput";

describe("ExpenseDateInput", () => {
    afterEach(cleanup);

    it("keeps the native date field as the accessible control and formats its visual value", () => {
        render(
            <ExpenseDateInput
                aria-label="Expense date"
                id="occurredOn"
                name="occurredOn"
                value="2026-09-02"
                onChange={vi.fn()}
            />
        );

        const nativeInput = screen.getByLabelText("Expense date");
        expect(nativeInput).toHaveAttribute("id", "occurredOn");
        expect(nativeInput).toHaveAttribute("type", "date");
        expect(nativeInput).toHaveValue("2026-09-02");
        expect(screen.getByDisplayValue("2026/09/02")).toHaveAttribute(
            "aria-hidden",
            "true"
        );
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("forwards changes from the native date field", () => {
        const onChange = vi.fn();

        render(
            <ExpenseDateInput
                aria-label="Expense date"
                name="occurredOn"
                value="2026-09-02"
                onChange={onChange}
            />
        );

        fireEvent.change(screen.getByLabelText("Expense date"), {
            target: { value: "2026-09-03" },
        });

        expect(onChange).toHaveBeenCalledOnce();
    });
});
