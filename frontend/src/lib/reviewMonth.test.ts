import { describe, expect, it } from "vitest";
import {
    formatReviewMonth,
    parseReviewMonth,
    previousClosedUTCMonth,
    shiftReviewMonth,
} from "./reviewMonth";

describe("review month helpers", () => {
    it("uses UTC and crosses year boundaries without parsing a date-only string", () => {
        expect(previousClosedUTCMonth(new Date("2026-01-01T00:30:00Z"))).toBe("2025-12");
        expect(shiftReviewMonth("2026-01", -1)).toBe("2025-12");
        expect(shiftReviewMonth("2026-12", 1)).toBe("2027-01");
        expect(formatReviewMonth("2026-08")).toBe("August 2026");
    });

    it("rejects non-canonical review months", () => {
        expect(parseReviewMonth("2026-8")).toBeNull();
        expect(parseReviewMonth("2026-13")).toBeNull();
        expect(parseReviewMonth("not-a-month")).toBeNull();
    });
});
