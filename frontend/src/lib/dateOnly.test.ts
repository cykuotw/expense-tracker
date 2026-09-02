import { describe, expect, it } from "vitest";
import {
    formatDateOnlyBrief,
    formatDateOnlyLong,
    isDateOnly,
    parseDateOnly,
    todayDateOnly,
} from "./dateOnly";

describe("dateOnly", () => {
    it("validates calendar dates without parsing them as instants", () => {
        expect(parseDateOnly("2024-02-29")).toEqual({ year: 2024, month: 2, day: 29 });
        expect(isDateOnly("2026-02-29")).toBe(false);
        expect(isDateOnly("2026-1-01")).toBe(false);
        expect(isDateOnly("2026-01-01T00:00:00Z")).toBe(false);
    });

    it("formats the same calendar day without timezone conversion", () => {
        expect(formatDateOnlyBrief("2026-01-01")).toEqual({ month: "Jan", day: "1" });
        expect(formatDateOnlyLong("2026-01-01")).toBe("Jan 1, 2026");
        expect(formatDateOnlyBrief("2026-12-31")).toEqual({ month: "Dec", day: "31" });
    });

    it("derives the default from local calendar components", () => {
        expect(todayDateOnly(new Date(2026, 0, 2, 23, 59))).toBe("2026-01-02");
    });
});
