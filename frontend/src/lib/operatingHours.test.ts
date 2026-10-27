import { describe, expect, it } from "vitest";
import { outsideOperatingHours, parseOperatingHours } from "./operatingHours";

const policy = { enabled: true, start: "09:00", stop: "22:00", safety_stop: "01:00", timezone: "America/Toronto" };

describe("operating-hours policy", () => {
    it("uses local daylight-saving time including both repeated 01:00 instants", () => {
        for (const time of ["2026-11-01T05:00:00Z", "2026-11-01T06:00:00Z", "2026-07-02T02:00:00Z"]) {
            expect(outsideOperatingHours(policy, new Date(time))).toBe(true);
        }
        for (const time of ["2026-07-01T13:00:00Z", "2026-01-01T14:00:00Z"]) {
            expect(outsideOperatingHours(policy, new Date(time))).toBe(false);
        }
    });
    it("rejects malformed public policy", () => {
        expect(parseOperatingHours(policy)).toEqual(policy);
        for (const value of [null, {}, { ...policy, timezone: "unknown" }, { ...policy, start: "25:00" }]) {
            expect(parseOperatingHours(value)).toBeNull();
        }
    });
});
