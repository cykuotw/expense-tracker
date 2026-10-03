import { describe, expect, it } from "vitest";
import { calculateAmount, normalizeCalculatorExpression } from "./amountCalculator";

describe("amount calculator", () => {
    it.each([
        ["12.50 + 8.25 + 3", 2, "23.75"],
        ["0.1 + 0.2", 2, "0.30"],
        ["10 - 2 * 3", 2, "4.00"],
        ["(10 - 2) × 3 ÷ 4", 2, "6.00"],
        ["1 / 3 * 3", 2, "1.00"],
        ["-2 + 5", 2, "3.00"],
        [".5 + 1.", 3, "1.500"],
        ["9007199254740992 + 1", 0, "9007199254740993"],
        [" 12.5 + 8.25 ", 2, "20.75"],
    ])("calculates %s exactly", (expression, digits, expected) => {
        expect(calculateAmount(expression as string, digits as number)).toEqual({ amount: expected, error: null, rounded: false });
    });

    it.each([
        ["1 / 3", 2, "0.33"],
        ["2 / 3", 2, "0.67"],
        ["1.005", 2, "1.01"],
        ["2.5", 0, "3"],
        ["1 / 6", 3, "0.167"],
    ])("rounds the final result of %s to currency precision", (expression, digits, expected) => {
        expect(calculateAmount(expression as string, digits as number)).toEqual({ amount: expected, error: null, rounded: true });
    });

    it.each(["1/0", "2/(1-1)", "1++", "2**", "1+-", "(2+3", "(1+)", "2 3", "3foo", "2**3", "alert(1)", "2;3", "0", "1-2", "0.001"])("rejects %s", (expression) => {
        expect(calculateAmount(expression, 2).amount).toBeNull();
        expect(calculateAmount(expression, 2).error).toBeTruthy();
    });

    it.each(["+", "-", "*", "/", "×", "÷", "−"])("ignores trailing %s without removing it while typing", (operator) => {
        const expression = `12 + 8 ${operator}  `;
        expect(normalizeCalculatorExpression(expression)).toBe("12 + 8");
        expect(calculateAmount(expression, 2).amount).toBe("20.00");
        expect(calculateAmount(`(12 + 8) ${operator}`, 2).amount).toBe("20.00");
        expect(calculateAmount(`1.${operator}`, 2).amount).toBe("1.00");
    });

    it("bounds input and requires currency metadata", () => {
        expect(calculateAmount("1".repeat(257), 2).amount).toBeNull();
        expect(calculateAmount("1+2", null).amount).toBeNull();
        expect(calculateAmount("", 2)).toEqual({ amount: null, error: null, rounded: false });
    });
});
