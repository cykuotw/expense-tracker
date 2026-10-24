import { useState } from "react";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { ConfirmedReceiptDetails } from "./ConfirmedReceiptDetails";
import type { ConfirmedExpenseItem } from "@/types/ocr";

const initialItems: ConfirmedExpenseItem[] = [
    { id: "one", description: "Coffee", quantity: "2", unit: "cup", unitPrice: "2.50", lineTotal: "5.00" },
    { id: "two", description: "Toast", quantity: "", unit: "", unitPrice: "", lineTotal: "4.00" },
];

function Harness({ collapseItems = false, validationMessage }: { collapseItems?: boolean; validationMessage?: string }) {
    const [items, setItems] = useState(initialItems);
    return <ConfirmedReceiptDetails
        merchant="Cafe" onMerchantChange={() => {}}
        subtotal="9.00" onSubtotalChange={() => {}}
        tax="" onTaxChange={() => {}}
        tip="" onTipChange={() => {}}
        items={items} onItemsChange={setItems} compactItems collapseItems={collapseItems} validationMessage={validationMessage}
    />;
}

afterEach(cleanup);

describe("Create Expense receipt items", () => {
    it("starts with compact item summaries and opens one item at a time", () => {
        render(<Harness />);

        expect(screen.getByLabelText("Subtotal (optional)")).toHaveAccessibleDescription("Leave blank to use total minus tax and tip.");
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

    it("folds edit items while retaining summary fields and unsaved item changes", () => {
        render(<Harness collapseItems />);
        const toggle = screen.getByRole("button", { name: "Items (2)" });
        expect(toggle).toHaveAttribute("aria-expanded", "false");
        expect(screen.getByLabelText("Merchant")).toHaveValue("Cafe");
        expect(screen.getByLabelText("Subtotal (optional)")).toBeVisible();
        expect(screen.queryByRole("button", { name: "Add item" })).not.toBeInTheDocument();
        fireEvent.click(toggle);
        expect(toggle).toHaveAttribute("aria-expanded", "true");
        fireEvent.click(screen.getByRole("button", { name: "Item 1: Coffee, 5.00" }));
        fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Latte" } });
        fireEvent.click(toggle);
        expect(screen.queryByLabelText("Description")).not.toBeInTheDocument();
        fireEvent.click(toggle);
        expect(screen.getByLabelText("Description")).toHaveValue("Latte");
        expect(screen.getByRole("button", { name: "Add item" })).toBeVisible();
    });

    it("keeps item validation visible while the list is collapsed", () => {
        render(<Harness collapseItems validationMessage="Item totals must equal subtotal." />);
        expect(screen.getByRole("button", { name: "Items (2)" })).toHaveAttribute("aria-expanded", "false");
        expect(screen.getByRole("alert")).toHaveTextContent("Item totals must equal subtotal.");
    });

});
