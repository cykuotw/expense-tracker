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

    it.each([
        { subtotal: "", tax: "", tip: "", total: "5.00", digits: 2, lineTotal: "5.00", expectedSubtotal: "5.00", expectedCharges: "0.00" },
        { subtotal: "   ", tax: "0", tip: "0", total: "5", digits: 2, lineTotal: "5", expectedSubtotal: "5.00", expectedCharges: "0.00" },
        { subtotal: "", tax: "0.65", tip: "1.00", total: "6.65", digits: 2, lineTotal: "5.00", expectedSubtotal: "5.00", expectedCharges: "1.65" },
        { subtotal: "", tax: "", tip: "", total: "500", digits: 0, lineTotal: "500", expectedSubtotal: "500", expectedCharges: "0" },
        { subtotal: "", tax: "0.123", tip: "", total: "5.123", digits: 3, lineTotal: "5.000", expectedSubtotal: "5.000", expectedCharges: "0.123" },
    ])("infers a blank subtotal with $digits currency digits and tax=$tax, tip=$tip", ({ digits, lineTotal, expectedSubtotal, expectedCharges, ...values }) => {
        expect(validateReceiptExpense({
            merchant: "Grocer", ...values, items: [item({ lineTotal })],
        }, digits)).toMatchObject({ valid: true, subTotal: expectedSubtotal, taxFeeTip: expectedCharges });
    });

    it.each([
        { subtotal: "", tax: "6.00", tip: "", total: "5.00" },
        { subtotal: "", tax: "-1.00", tip: "", total: "4.00" },
        { subtotal: "", tax: "invalid", tip: "", total: "5.00" },
        { subtotal: "", tax: "", tip: "", total: "5.001" },
        { subtotal: "", tax: "", tip: "", total: "4.00" },
        { subtotal: "0", tax: "", tip: "", total: "5.00" },
    ])("rejects invalid or inconsistent amounts even when subtotal is optional: %o", values => {
        expect(validateReceiptExpense({ merchant: "Grocer", ...values, items: [item()] }, 2).valid).toBe(false);
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
