import { describe, expect, it } from "vitest";
import { notificationDestination } from "./notificationNavigation";

describe("notificationDestination", () => {
    it("keeps monthly-review notification navigation on the application origin", () => {
        expect(
            notificationDestination(
                "https://expenses.example.com",
                "/group/group-1/monthly-review/2026-08",
            ),
        ).toBe("https://expenses.example.com/group/group-1/monthly-review/2026-08");
        expect(
            notificationDestination(
                "https://expenses.example.com",
                "https://attacker.example/review",
            ),
        ).toBe("https://expenses.example.com/");
        expect(notificationDestination("https://expenses.example.com", undefined)).toBe(
            "https://expenses.example.com/",
        );
        expect(notificationDestination("https://expenses.example.com", "https://%"))
            .toBe("https://expenses.example.com/");
    });
});
