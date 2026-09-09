import { describe, expect, it } from "vitest";
import { decimalToUnits, normalizeMoney, splitEqually, unitsToDecimal } from "./money";

describe("money", () => {
    it("converts decimal strings without floating point", () => {
        expect(decimalToUnits("0.30", 2)).toBe(30n);
        expect(unitsToDecimal(30n, 2)).toBe("0.30");
        expect(normalizeMoney("1.230", 2)).toBe("1.23");
    });

    it("rejects unsupported decimal syntax and precision", () => {
        expect(decimalToUnits(".5", 2)).toBeNull();
        expect(decimalToUnits("1e2", 2)).toBeNull();
        expect(decimalToUnits("100.01", 0)).toBeNull();
    });

    it("allocates equal-split remainders by stable user id", () => {
        const shares = splitEqually(1000n, ["user-c", "user-a", "user-b"]);
        expect(shares?.get("user-a")).toBe(334n);
        expect(shares?.get("user-b")).toBe(333n);
        expect(shares?.get("user-c")).toBe(333n);
    });
});
