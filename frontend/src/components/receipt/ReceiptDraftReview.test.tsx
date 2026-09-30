import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

const { useCreateExpenseMock } = vi.hoisted(() => ({ useCreateExpenseMock: vi.fn() }));
vi.mock("@/hooks/CreateExpenseContextHooks", () => ({
    useCreateExpense: () => useCreateExpenseMock(),
}));

import ReceiptDraftReview from "./ReceiptDraftReview";

const provider = (value: string, requiresReview = false) => ({
    value,
    confidence: 92,
    provenance: "provider" as const,
    requiresReview,
});
const draft = {
    merchant: provider("Cafe", true),
    date: provider("2026-09-27", true),
    currencySuggestion: provider("USD", true),
    subtotal: provider("10.00"),
    tax: { value: "0", provenance: "inferred" as const, requiresReview: true },
    tip: provider(""),
    total: provider("10.00"),
    items: [
        { description: provider("Coffee"), quantity: { value: "1", provenance: "inferred" as const, requiresReview: true }, unitPrice: provider("4.00"), lineTotal: provider("4.00") },
        { description: provider("Cake"), quantity: provider("1"), unitPrice: provider("6.00"), lineTotal: provider("6.00") },
    ],
};

function renderReview(openBreakdown = true) {
    const result = render(
        <MemoryRouter initialEntries={["/create_expense/receipt/review?g=group-1"]}>
            <Routes>
                <Route path="/create_expense/receipt/review" element={<ReceiptDraftReview />} />
                <Route path="/create_expense" element={<div>Manual form</div>} />
            </Routes>
        </MemoryRouter>,
    );
    const breakdown = screen.queryByText("Breakdown");
    if (openBreakdown && breakdown) fireEvent.click(breakdown.closest("summary")!);
    return result;
}

describe("ReceiptDraftReview", () => {
    const applyReviewedReceipt = vi.fn();
    const clearReceiptWorkflow = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        useCreateExpenseMock.mockReturnValue({
            selectedGroupId: "group-1",
            currency: "CAD",
            amountDigits: 2,
            occurredOn: "2026-09-28",
            ocrStatus: "ready",
            ocrDraft: draft,
            ocrError: null,
            applyReviewedReceipt,
            clearReceiptWorkflow,
        });
    });

    afterEach(cleanup);

    it("labels extracted and inferred values, warns on currency mismatch, and applies edited item order", () => {
        renderReview();

        expect(screen.queryByText("Extracted")).not.toBeInTheDocument();
        expect(screen.getAllByText("Inferred").length).toBeGreaterThan(0);
        expect(screen.getByRole("heading", { name: "Key details" })).toBeInTheDocument();
        expect(screen.getByText(/Receipt says USD/)).toBeInTheDocument();
        expect(screen.getByLabelText(/^Total/)).toHaveValue(10);

        fireEvent.change(screen.getByLabelText(/Merchant/), { target: { value: "Edited Cafe" } });
        fireEvent.click(screen.getByRole("button", { name: "Move item 2 up" }));
        fireEvent.click(screen.getByRole("button", { name: /Use reviewed values/i }));

        expect(applyReviewedReceipt).toHaveBeenCalledWith(expect.objectContaining({
            merchant: "Edited Cafe",
            currencySuggestion: "USD",
            items: [
                expect.objectContaining({ description: "Cake", lineTotal: "6.00" }),
                expect.objectContaining({ description: "Coffee", lineTotal: "4.00" }),
            ],
        }));
        expect(screen.getByText("Manual form")).toBeInTheDocument();
    });

    it("uses the expense date as an inferred fallback when OCR has no date", () => {
        useCreateExpenseMock.mockReturnValue({
            ...useCreateExpenseMock(),
            ocrDraft: { ...draft, date: provider("") },
        });
        renderReview();
        expect(screen.getByLabelText(/^Date/)).toHaveValue("2026-09-28");
        expect(screen.getAllByText("Inferred").length).toBeGreaterThan(0);
        expect(screen.getByText(/No receipt date was found/)).toBeInTheDocument();
    });

    it("starts with the breakdown collapsed and opens item controls on demand", () => {
        renderReview(false);
        expect(screen.getByText("Breakdown").closest("details")).not.toHaveAttribute("open");
        fireEvent.click(screen.getByText("Breakdown").closest("summary")!);
        expect(screen.getByRole("group", { name: "Item 1" })).toBeInTheDocument();
    });

    it("collapses secondary item fields behind a native disclosure", () => {
        renderReview();
        const summary = screen.getAllByText("Quantity, unit & price")[0].closest("summary");
        const details = summary?.closest("details");
        expect(details).not.toHaveAttribute("open");
        fireEvent.click(summary!);
        expect(details).toHaveAttribute("open");
    });

    it("keeps review cues visible for flagged fields and inferred item quantity", () => {
        renderReview();
        const receipt = screen.getByRole("region", { name: "Key details" });
        expect(within(receipt).getAllByText("Check")).toHaveLength(2);
        expect(within(screen.getByRole("region", { name: "Amounts" })).getByText("Check")).toBeInTheDocument();

        const item = screen.getByRole("group", { name: "Item 1" });
        expect(within(item).getByText("Check inferred quantity")).toBeInTheDocument();
        fireEvent.click(within(item).getByText("Quantity, unit & price"));
        fireEvent.change(within(item).getByLabelText("Quantity"), { target: { value: "2" } });
        expect(within(item).queryByText("Check inferred quantity")).not.toBeInTheDocument();
    });

    it("shows item validation with Items and amount validation with Amounts", () => {
        renderReview();
        const items = screen.getByRole("region", { name: /Items/ });
        const amounts = screen.getByRole("region", { name: "Amounts" });
        fireEvent.change(within(screen.getByRole("group", { name: "Item 1" })).getByLabelText("Line total"), { target: { value: "5.00" } });
        expect(within(items).getByRole("alert")).toHaveTextContent("Item line totals must add up to the subtotal");
        expect(within(amounts).queryByRole("alert")).not.toBeInTheDocument();

        fireEvent.change(screen.getByLabelText(/^Total/), { target: { value: "11.00" } });
        expect(within(amounts).getByRole("alert")).toHaveTextContent("Subtotal, tax, and tip must add up to the total");
        expect(within(items).queryByRole("alert")).not.toBeInTheDocument();
    });

    it("keeps secondary edits through collapse and reorder, and blocks invalid hidden values", () => {
        renderReview();
        const item = screen.getByRole("group", { name: "Item 1" });
        const disclosure = within(item).getByText("Quantity, unit & price");
        fireEvent.click(disclosure);
        fireEvent.change(within(item).getByLabelText("Quantity"), { target: { value: "0" } });
        fireEvent.click(disclosure);
        expect(screen.getByRole("button", { name: /Use reviewed values/i })).toBeDisabled();
        expect(screen.getByRole("alert")).toHaveTextContent("Item quantity must be positive");
        fireEvent.click(disclosure);
        fireEvent.change(within(item).getByLabelText("Quantity"), { target: { value: "2" } });
        fireEvent.change(within(item).getByLabelText("Unit"), { target: { value: "cups" } });
        fireEvent.change(within(item).getByLabelText("Unit price"), { target: { value: "2.00" } });
        fireEvent.click(disclosure);
        fireEvent.click(screen.getByRole("button", { name: "Move item 1 down" }));
        fireEvent.click(screen.getByRole("button", { name: /Use reviewed values/i }));
        expect(applyReviewedReceipt.mock.calls[0][0].items[1]).toEqual(expect.objectContaining({
            description: "Coffee", quantity: "2", unit: "cups", unitPrice: "2.00", lineTotal: "4.00",
        }));
    });

    it("validates a newly added item and applies it after correction", () => {
        renderReview();
        fireEvent.click(screen.getByRole("button", { name: "Add item" }));
        expect(screen.getByRole("button", { name: /Use reviewed values/i })).toBeDisabled();
        const item = screen.getByRole("group", { name: "Item 3" });
        fireEvent.change(within(item).getByLabelText("Description"), { target: { value: "Included extra" } });
        fireEvent.change(within(item).getByLabelText("Line total"), { target: { value: "0" } });
        fireEvent.click(screen.getByRole("button", { name: /Use reviewed values/i }));
        expect(applyReviewedReceipt.mock.calls[0][0].items).toHaveLength(3);
    });

    it("supports item removal without persisting the discarded suggestion", () => {
        renderReview();
        fireEvent.click(screen.getByRole("button", { name: "Remove item 1" }));
        fireEvent.change(screen.getByLabelText(/^Subtotal/), { target: { value: "6.00" } });
        fireEvent.change(screen.getByLabelText(/^Total/), { target: { value: "6.00" } });
        fireEvent.click(screen.getByRole("button", { name: /Use reviewed values/i }));
        expect(applyReviewedReceipt.mock.calls[0][0].items).toEqual([
            expect.objectContaining({ description: "Cake" }),
        ]);
    });

    it("offers safe manual recovery after OCR failure", () => {
        useCreateExpenseMock.mockReturnValue({
            selectedGroupId: "group-1",
            currency: "CAD",
            amountDigits: 2,
            occurredOn: "2026-09-28",
            ocrStatus: "error",
            ocrDraft: null,
            ocrError: "Receipt scanning timed out.",
            applyReviewedReceipt,
            clearReceiptWorkflow,
        });
        renderReview();
        expect(screen.getByRole("alert")).toHaveTextContent("Receipt scanning timed out.");
        fireEvent.click(screen.getByRole("button", { name: /Continue manually/i }));
        expect(clearReceiptWorkflow).toHaveBeenCalledOnce();
        expect(screen.getByText("Manual form")).toBeInTheDocument();
    });
});
