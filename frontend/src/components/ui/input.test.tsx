import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Input } from "./input";

describe("Input", () => {
    it("applies the expense form treatment through its expense variant", () => {
        render(<Input aria-label="Expense date" type="text" variant="expense" />);

        const input = screen.getByLabelText("Expense date");

        expect(input).toHaveClass(
            "expense-form-input-shell",
            "rounded-2xl",
            "border-border",
            "bg-background",
            "px-4"
        );
    });
});
