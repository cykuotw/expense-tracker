import { decimalToUnits, unitsToDecimal } from "./money";
import type { ExpenseCategoryAmount } from "../types/expense";

export function calculateCategoryBreakdown(rows: ExpenseCategoryAmount[], total: string, digits: number | null) {
    if (rows.length === 0) return { valid: true, remaining: null, error: null };
    if (digits === null) return { valid: false, remaining: null, error: "Choose a currency to enter category amounts." };
    const totalUnits = decimalToUnits(total, digits);
    const amounts = rows.map(row => decimalToUnits(row.amount, digits));
    const categorized = amounts.reduce<bigint>((sum, amount) => sum + (amount ?? 0n), 0n);
    const remaining = totalUnits === null ? null : totalUnits - categorized;
    let error: string | null = null;
    if (rows.length < 2 || rows.length > 20) error = "Use 2–20 categories, or choose Use one category.";
    else if (rows.some(row => !row.expenseTypeId)) error = "Choose a category for each amount.";
    else if (new Set(rows.map(row => row.expenseTypeId)).size !== rows.length) error = "Combine amounts with the same category into one row.";
    else if (amounts.some(amount => amount === null || amount <= 0n)) error = "Enter a positive amount for each category using this currency's precision.";
    else if (rows.some(row => Array.from(row.note ?? "").length > 200)) error = "Keep each note to 200 characters.";
    else if (totalUnits === null || totalUnits <= 0n) error = "Enter the expense total first.";
    else if (remaining !== 0n) error = remaining! > 0n ? "Categorize the remaining amount before saving." : "Reduce category amounts to match the expense total.";
    return { valid: error === null, remaining, error };
}

export function categoryBreakdownPayload(rows: ExpenseCategoryAmount[], digits: number): ExpenseCategoryAmount[] {
    return rows.map(({ expenseTypeId, amount, note }) => ({
        expenseTypeId, amount: unitsToDecimal(decimalToUnits(amount, digits)!, digits),
        ...(note ? { note } : {}),
    }));
}
