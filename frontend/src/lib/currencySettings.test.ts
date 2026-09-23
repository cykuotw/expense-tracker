import { afterEach, describe, expect, it, vi } from "vitest";
import {
    compactRate,
    divideRates,
    fetchRateRecommendations,
    reciprocalRate,
    validPositiveRate,
} from "./currencySettings";

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

describe("rate recommendations", () => {
    afterEach(() => vi.unstubAllGlobals());

    it("fetches public quotes without credentials and returns canonical rates", async () => {
        const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([
            { date: "2026-09-22", base: "CAD", quote: "TWD", rate: 23.25581395348837 },
            { date: "2026-09-22", base: "CAD", quote: "USD", rate: 0.7407407407407407 },
        ]), { status: 200, headers: { "Content-Type": "application/json" } }));
        vi.stubGlobal("fetch", fetchMock);

        const result = await fetchRateRecommendations(["USD", "CAD", "TWD"], "CAD");

        const request = new URL(String(fetchMock.mock.calls[0][0]));
        expect(request.origin).toBe("https://api.frankfurter.dev");
        expect(request.searchParams.get("base")).toBe("CAD");
        expect(request.searchParams.get("quotes")).toBe("TWD,USD");
        expect(fetchMock.mock.calls[0][1]).toEqual(expect.objectContaining({ credentials: "omit" }));
        expect(result.recommendations).toEqual([
            expect.objectContaining({ sourceCurrency: "CAD", recommendedRate: "1" }),
            expect.objectContaining({ sourceCurrency: "TWD", recommendedRate: "0.043" }),
            expect.objectContaining({ sourceCurrency: "USD", recommendedRate: "1.35" }),
        ]);
    });

    it("rejects incomplete provider responses", async () => {
        vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("[]", { status: 200 })));

        await expect(fetchRateRecommendations(["CAD", "TWD"], "CAD"))
            .rejects.toThrow("missing recommendation snapshot");
    });
});
