import { useState } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ExpenseCategoryEditor } from "./ExpenseCategoryEditor";
import type { ExpenseCategoryAmount } from "../../types/expense";

const types = [
    { id: "groceries", name: "Groceries", category: "Food and Drink" },
    { id: "household", name: "Household Supplies", category: "Home" },
    { id: "clothing", name: "Clothing", category: "Shopping" },
];

function Harness({ initial = [] }: { initial?: ExpenseCategoryAmount[] }) {
    const [rows, setRows] = useState(initial);
    const [type, setType] = useState("groceries");
    const [total, setTotal] = useState("300");
    return <><label>Total<input aria-label="Total" value={total} onChange={event => setTotal(event.target.value)} /></label>
        <ExpenseCategoryEditor rows={rows} onChange={setRows} expenseTypes={types} singleTypeId={type} onSingleTypeChange={setType} total={total} currency="CAD" amountDigits={2} />
        <output aria-label="Saved rows">{JSON.stringify(rows)}</output></>;
}

afterEach(cleanup);

describe("category amounts", () => {
    it("completes the Costco flow with an explicit remainder and a third category", async () => {
        render(<Harness />);
        expect(screen.queryByLabelText("Amount for category 1 (CAD)")).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: "Split across categories" }));
        expect(screen.getByRole("status", { name: "Category amount status" })).not.toHaveTextContent("All CAD 300.00 categorized");
        const first = screen.getByLabelText<HTMLInputElement>("Amount for category 1 (CAD)");
        await waitFor(() => expect(first).toHaveFocus());
        expect(first.selectionStart).toBe(0);
        expect(first.selectionEnd).toBe(3);
        fireEvent.change(first, { target: { value: "200" } });
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("CAD 100.00 left to categorize");
        fireEvent.click(screen.getByRole("button", { name: "Category 2: Choose a category" }));
        fireEvent.click(screen.getByRole("option", { name: "Household Supplies" }));
        fireEvent.click(screen.getByRole("button", { name: "Use remaining CAD 100.00" }));
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("All CAD 300.00 categorized");
        expect(JSON.parse(screen.getByLabelText("Saved rows").textContent!).map(({ expenseTypeId, amount }: ExpenseCategoryAmount) => ({ expenseTypeId, amount }))).toEqual([
            { expenseTypeId: "groceries", amount: "200" }, { expenseTypeId: "household", amount: "100.00" },
        ]);
        fireEvent.click(screen.getByRole("button", { name: "Add category" }));
        expect(screen.getByLabelText("Amount for category 3 (CAD)")).toBeInTheDocument();
        fireEvent.change(screen.getByLabelText("Total"), { target: { value: "350" } });
        expect(first).toHaveValue("200");
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("CAD 50.00 left to categorize");
        fireEvent.click(screen.getByRole("button", { name: "Category 3: Choose a category" }));
        fireEvent.click(screen.getByRole("option", { name: "Clothing" }));
        fireEvent.click(screen.getByRole("button", { name: "Use remaining CAD 50.00" }));
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("All CAD 350.00 categorized");
    });

    it.each([
        { amount: "200", note: "Food" },
        { amount: "200", note: undefined },
        { amount: "300", note: "Food" },
    ])("confirms removing an empty row when the retained amount/note would change: %j", ({ amount, note }) => {
        const initial = [{ expenseTypeId: "groceries", amount, ...(note ? { note } : {}) }, { expenseTypeId: "", amount: "" }];
        render(<Harness initial={initial} />);
        fireEvent.click(screen.getByRole("button", { name: "Remove category 2" }));
        expect(screen.getByRole("button", { name: "Use this category for the full total" })).toBeVisible();
        expect(JSON.parse(screen.getByLabelText("Saved rows").textContent!)).toEqual(initial);
        fireEvent.click(screen.getByRole("button", { name: "Keep category amounts" }));
        expect(screen.getByLabelText("Amount for category 1 (CAD)")).toHaveValue(amount);
        if (note) expect(screen.getByDisplayValue(note)).toBeVisible();
        fireEvent.click(screen.getByRole("button", { name: "Remove category 2" }));
        fireEvent.click(screen.getByRole("button", { name: "Use this category for the full total" }));
        expect(JSON.parse(screen.getByLabelText("Saved rows").textContent!)).toEqual([]);
    });

    it("collapses an untouched full-total row without a redundant confirmation", () => {
        render(<Harness initial={[{ expenseTypeId: "groceries", amount: "300" }, { expenseTypeId: "", amount: "" }]} />);
        fireEvent.click(screen.getByRole("button", { name: "Remove category 2" }));
        expect(screen.queryByRole("button", { name: "Use this category for the full total" })).toBeNull();
        expect(screen.getByRole("button", { name: "Split across categories" })).toBeVisible();
    });

    it("describes category and amount errors independently when both fields are invalid", () => {
        render(<Harness initial={[{ expenseTypeId: "groceries", amount: "200" }, { expenseTypeId: "", amount: "-1" }]} />);
        const amount = screen.getByLabelText("Amount for category 2 (CAD)");
        fireEvent.blur(amount);
        const category = screen.getByRole("button", { name: "Category 2: Choose a category" });
        expect(category).toHaveAttribute("aria-invalid", "true");
        expect(category).toHaveAccessibleDescription("Choose a category.");
        expect(amount).toHaveAttribute("aria-invalid", "true");
        expect(amount).toHaveAccessibleDescription("Enter an amount greater than zero with up to 2 decimal places.");
        expect(category.getAttribute("aria-describedby")).not.toBe(amount.getAttribute("aria-describedby"));
        fireEvent.click(category);
        fireEvent.click(screen.getByRole("option", { name: "Household Supplies" }));
        expect(amount).toHaveAccessibleDescription("Enter an amount greater than zero with up to 2 decimal places.");
        fireEvent.change(amount, { target: { value: "100" } });
        expect(amount).not.toHaveAttribute("aria-invalid");
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("All CAD 300.00 categorized");
    });

    it("shows actionable excess/precision feedback and preserves entered notes on cancelled conversion", () => {
        render(<Harness initial={[{ expenseTypeId: "groceries", amount: "200", note: "Food" }, { expenseTypeId: "household", amount: "120" }]} />);
        expect(screen.getByRole("status", { name: "Category amount status" })).toHaveTextContent("CAD 20.00 over the total");
        fireEvent.change(screen.getByLabelText("Amount for category 2 (CAD)"), { target: { value: "100.001" } });
        fireEvent.blur(screen.getByLabelText("Amount for category 2 (CAD)"));
        expect(screen.getByLabelText("Amount for category 2 (CAD)")).toHaveAttribute("aria-invalid", "true");
        expect(screen.getByText(/Enter an amount greater than zero/)).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "Use one category" }));
        fireEvent.click(screen.getByRole("button", { name: "Keep category amounts" }));
        expect(screen.getByDisplayValue("Food")).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "Use one category" }));
        fireEvent.click(screen.getByRole("button", { name: "Use this category for the full total" }));
        expect(screen.getByLabelText("Saved rows")).toHaveTextContent("[]");
        expect(screen.getByRole("button", { name: "Split across categories" })).toBeInTheDocument();
    });
});
