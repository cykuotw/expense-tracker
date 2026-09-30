import { useState } from "react";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { ConfirmedReceiptDetails } from "./ConfirmedReceiptDetails";
import type { ConfirmedExpenseItem } from "@/types/ocr";

const initialItems: ConfirmedExpenseItem[] = [
    { id: "one", description: "Coffee", quantity: "2", unit: "cup", unitPrice: "2.50", lineTotal: "5.00" },
    { id: "two", description: "Toast", quantity: "", unit: "", unitPrice: "", lineTotal: "4.00" },
];

function Harness() {
    const [items, setItems] = useState(initialItems);
    return <ConfirmedReceiptDetails
        merchant="Cafe" onMerchantChange={() => {}}
        subtotal="9.00" onSubtotalChange={() => {}}
        tax="" onTaxChange={() => {}}
        tip="" onTipChange={() => {}}
        items={items} onItemsChange={setItems} compactItems
    />;
}

afterEach(cleanup);

describe("Create Expense receipt items", () => {
    it("starts with compact item summaries and opens one item at a time", () => {
        render(<Harness />);

        expect(screen.getByRole("button", { name: "Item 1: Coffee, 5.00" })).toHaveAttribute("aria-expanded", "false");
        expect(screen.queryByLabelText("Description")).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Item 1: Coffee, 5.00" }));
        expect(screen.getByRole("button", { name: "Item 1: Coffee, 5.00" })).toHaveAttribute("aria-expanded", "true");
        fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Latte" } });
        expect(screen.getByRole("button", { name: "Item 1: Latte, 5.00" })).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Item 2: Toast, 4.00" }));
        expect(screen.getByRole("button", { name: "Item 1: Latte, 5.00" })).toHaveAttribute("aria-expanded", "false");
        expect(screen.getByLabelText("Description")).toHaveValue("Toast");
    });

    it("opens a newly added item for immediate editing", () => {
        render(<Harness />);
        fireEvent.click(screen.getByRole("button", { name: "Add item" }));
        const item = screen.getByRole("group", { name: "Item 3" });
        expect(within(item).getByRole("button", { name: "Item 3: Add description, Add total" })).toHaveAttribute("aria-expanded", "true");
        expect(within(item).getByLabelText("Description")).toHaveValue("");
    });
});
