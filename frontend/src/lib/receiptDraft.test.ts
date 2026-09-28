import { describe, expect, it } from "vitest";
import { validateReceiptExpense } from "./receiptDraft";

const item = (overrides = {}) => ({
    id: "item-1",
    description: "Coffee",
    quantity: "2",
    unit: "cup",
    unitPrice: "2.50",
    lineTotal: "5.00",
    ...overrides,
});

describe("validateReceiptExpense", () => {
    it("maps reviewed values to the confirmed Worker contract", () => {
        expect(validateReceiptExpense({
            merchant: "Cafe",
            subtotal: "5.00",
            tax: "0.65",
            tip: "1.00",
            total: "6.65",
            items: [item()],
        }, 2)).toEqual({
            valid: true,
            message: null,
            subTotal: "5.00",
            taxFeeTip: "1.65",
            items: [{
                description: "Coffee",
                quantity: "2",
                unit: "cup",
                unitPrice: "2.50",
                lineTotal: "5.00",
            }],
        });
    });

    it("rejects totals that do not match the reviewed breakdown", () => {
        const result = validateReceiptExpense({
            merchant: "Cafe",
            subtotal: "5.00",
            tax: "0.65",
            tip: "1.00",
            total: "7.00",
            items: [item()],
        }, 2);
        expect(result.valid).toBe(false);
        expect(result.message).toMatch(/add up to the total/i);
    });

    it("rejects item totals that do not match subtotal", () => {
        const result = validateReceiptExpense({
            merchant: "Cafe",
            subtotal: "6.00",
            tax: "0",
            tip: "0",
            total: "6.00",
            items: [item()],
        }, 2);
        expect(result.valid).toBe(false);
        expect(result.message).toMatch(/line totals/i);
    });

    it("keeps the existing manual expense path valid without a breakdown", () => {
        expect(validateReceiptExpense({
            merchant: "",
            subtotal: "",
            tax: "",
            tip: "",
            total: "12.34",
            items: [],
        }, 2)).toEqual({ valid: true, message: null, items: [] });
    });
});
