import { describe, expect, it } from "vitest";
import { compactRate, divideRates, reciprocalRate, validPositiveRate } from "./currencySettings";

describe("currency rate precision", () => {
    it("accepts values that fit NUMERIC(30,15)", () => {
        expect(validPositiveRate("999999999999999.999999999999999")).toBe(true);
        expect(validPositiveRate("0.000000000000001")).toBe(true);
    });

    it("rejects values outside NUMERIC(30,15)", () => {
        expect(validPositiveRate("1000000000000000")).toBe(false);
        expect(validPositiveRate("1.0000000000000001")).toBe(false);
        expect(validPositiveRate("0")).toBe(false);
        expect(validPositiveRate("1e-3")).toBe(false);
    });

    it("rounds reciprocal and cross rates to 15 fractional digits", () => {
        expect(reciprocalRate("1.35")).toBe("0.740740740740741");
        expect(reciprocalRate("25")).toBe("0.04");
        expect(reciprocalRate("0.000000000000001")).toBeNull();
        expect(divideRates("1.35", "0.04")).toBe("33.75");
    });

    it("presents recommendations with eight significant digits", () => {
        expect(compactRate("23.255813953488372")).toBe("23.255814");
        expect(compactRate("0.000012345678912")).toBe("0.000012345679");
        expect(compactRate("123456789.123456789")).toBe("123456789");
    });
});
