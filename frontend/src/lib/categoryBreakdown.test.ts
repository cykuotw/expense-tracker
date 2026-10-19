import { describe, expect, it } from "vitest";
import { calculateCategoryBreakdown, categoryBreakdownPayload } from "./categoryBreakdown";
const rows = [{ expenseTypeId: "a", amount: "0.10" }, { expenseTypeId: "b", amount: "0.20" }];
describe("category amount validation", () => {
    it("uses exact money and supports legacy single-category expenses", () => {
        expect(calculateCategoryBreakdown([], "", null).valid).toBe(true);
        expect(calculateCategoryBreakdown(rows, "0.30", 2)).toEqual({ valid: true, remaining: 0n, error: null });
        expect(categoryBreakdownPayload(rows, 2)).toEqual(rows);
    });
    it("rejects duplicates, invalid precision, nonpositive values, invalid counts and long notes", () => {
        for (const invalid of [[rows[0]], [rows[0], rows[0]], [{ ...rows[0], amount: "0" }, rows[1]],
            [{ ...rows[0], amount: "-1" }, rows[1]], [{ ...rows[0], amount: "0.001" }, rows[1]],
            [{ ...rows[0], note: "x".repeat(201) }, rows[1]], Array.from({ length: 21 }, (_, i) => ({ expenseTypeId: `${i}`, amount: "1" }))]) {
            expect(calculateCategoryBreakdown(invalid, "0.30", 2).valid).toBe(false);
        }
        expect(calculateCategoryBreakdown(rows, "0.30", 0).valid).toBe(false);
        expect(calculateCategoryBreakdown([{ expenseTypeId: "a", amount: "1" }, { expenseTypeId: "b", amount: "2" }], "3", 0).valid).toBe(true);
    });
});
