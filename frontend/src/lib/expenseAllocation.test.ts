import { describe, expect, it } from "vitest";
import {
    basisPointsToPercentage,
    calculateExpenseAllocation,
    percentageInputToBasisPoints,
} from "./expenseAllocation";

const users = ["user-c", "user-a", "user-b"];

describe("expense allocation", () => {
    it("splits equal remainders by stable participant id", () => {
        const result = calculateExpenseAllocation(
            "10.00",
            2,
            {
                mode: "equal",
                participants: users.map((userId) => ({ userId })),
            },
            "CAD"
        );

        expect(result.valid).toBe(true);
        expect(result.shares.get("user-a")).toBe(334n);
        expect(result.shares.get("user-b")).toBe(333n);
        expect(result.shares.get("user-c")).toBe(333n);
    });

    it("requires exact amounts to match the total", () => {
        const result = calculateExpenseAllocation(
            "10.000",
            3,
            {
                mode: "exact",
                participants: [
                    { userId: "user-a", amount: "4.001" },
                    { userId: "user-b", amount: "5.999" },
                ],
            },
            "KWD"
        );

        expect(result.valid).toBe(true);
        expect(result.shares.get("user-a")).toBe(4001n);
        expect(result.shares.get("user-b")).toBe(5999n);
    });

    it("uses largest remainder and user id ties for percentages", () => {
        const result = calculateExpenseAllocation(
            "0.05",
            2,
            {
                mode: "percentage",
                participants: [
                    { userId: "user-a", percentageBasisPoints: 3333 },
                    { userId: "user-b", percentageBasisPoints: 3333 },
                    { userId: "user-c", percentageBasisPoints: 3334 },
                ],
            },
            "CAD"
        );

        expect(result.valid).toBe(true);
        expect(result.shares.get("user-a")).toBe(2n);
        expect(result.shares.get("user-b")).toBe(1n);
        expect(result.shares.get("user-c")).toBe(2n);
    });

    it("adds adjustments after splitting the remainder equally", () => {
        const result = calculateExpenseAllocation(
            "10",
            0,
            {
                mode: "adjustment",
                participants: [
                    { userId: "user-a", amount: "1" },
                    { userId: "user-b", amount: "0" },
                    { userId: "user-c" },
                ],
            },
            "JPY"
        );

        expect(result.valid).toBe(true);
        expect(result.shares.get("user-a")).toBe(4n);
        expect(result.shares.get("user-b")).toBe(3n);
        expect(result.shares.get("user-c")).toBe(3n);
    });

    it("rejects invalid percentage inputs without floating-point conversion", () => {
        expect(percentageInputToBasisPoints("33.33")).toBe(3333);
        expect(percentageInputToBasisPoints("33.333")).toBeNull();
        expect(percentageInputToBasisPoints("100.01")).toBeNull();
        expect(basisPointsToPercentage(3330)).toBe("33.3");
    });

    it("keeps selected zero-value participants distinct", () => {
        const result = calculateExpenseAllocation(
            "5.00",
            2,
            {
                mode: "exact",
                participants: [
                    { userId: "user-a", amount: "5" },
                    { userId: "user-b", amount: "0" },
                ],
            },
            "CAD"
        );

        expect(result.valid).toBe(true);
        expect(result.shares.has("user-b")).toBe(true);
        expect(result.shares.get("user-b")).toBe(0n);
    });
});
