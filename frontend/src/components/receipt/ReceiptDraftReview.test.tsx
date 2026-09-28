import { cleanup, fireEvent, render, screen } from "@testing-library/react";
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
        { description: provider("Coffee"), quantity: { value: "1", provenance: "inferred" as const }, unitPrice: provider("4.00"), lineTotal: provider("4.00") },
        { description: provider("Cake"), quantity: provider("1"), unitPrice: provider("6.00"), lineTotal: provider("6.00") },
    ],
};

function renderReview() {
    return render(
        <MemoryRouter initialEntries={["/create_expense/receipt/review?g=group-1"]}>
            <Routes>
                <Route path="/create_expense/receipt/review" element={<ReceiptDraftReview />} />
                <Route path="/create_expense" element={<div>Manual form</div>} />
            </Routes>
        </MemoryRouter>,
    );
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

        expect(screen.getAllByText("Extracted").length).toBeGreaterThan(0);
        expect(screen.getAllByText("Inferred").length).toBeGreaterThan(0);
        expect(screen.getAllByText("Check this")).toHaveLength(2);
        expect(screen.getByText(/receipt suggested USD/i)).toBeInTheDocument();

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
