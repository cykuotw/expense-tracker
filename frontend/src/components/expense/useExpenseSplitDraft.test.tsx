import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ExpenseAllocation } from "../../types/allocation";
import { useExpenseSplitDraft } from "./useExpenseSplitDraft";

const members = [
    { userId: "user-a", username: "Alice" },
    { userId: "user-b", username: "Bob" },
];

function renderDraft(allocation: ExpenseAllocation, total = "10.00") {
    return renderHook(
        ({ currentTotal }) =>
            useExpenseSplitDraft({
                allocation,
                amountDigits: 2,
                currency: "CAD",
                members,
                total: currentTotal,
            }),
        { initialProps: { currentTotal: total } },
    );
}

describe("useExpenseSplitDraft", () => {
    it("keeps exact inputs and revalidates them when the total changes", () => {
        const hook = renderDraft({
            mode: "exact",
            participants: [
                { userId: "user-a", amount: "6.00" },
                { userId: "user-b", amount: "4.00" },
            ],
        });

        expect(hook.result.current.calculation.valid).toBe(true);
        hook.rerender({ currentTotal: "12.00" });

        expect(hook.result.current.draftAllocation.participants).toEqual([
            { userId: "user-a", amount: "6.00" },
            { userId: "user-b", amount: "4.00" },
        ]);
        expect(hook.result.current.calculation.valid).toBe(false);
    });

    it("recalculates percentage shares when the total changes", () => {
        const hook = renderDraft({
            mode: "percentage",
            participants: [
                { userId: "user-a", percentageBasisPoints: 6000 },
                { userId: "user-b", percentageBasisPoints: 4000 },
            ],
        });

        hook.rerender({ currentTotal: "20.00" });

        expect(hook.result.current.calculation.valid).toBe(true);
        expect(hook.result.current.calculation.shares.get("user-a")).toBe(
            1200n,
        );
        expect(hook.result.current.calculation.shares.get("user-b")).toBe(
            800n,
        );
    });

    it("keeps adjustments and redistributes the new remainder", () => {
        const hook = renderDraft({
            mode: "adjustment",
            participants: [
                { userId: "user-a", amount: "1.00" },
                { userId: "user-b", amount: "0" },
            ],
        });

        hook.rerender({ currentTotal: "12.00" });

        expect(hook.result.current.calculation.valid).toBe(true);
        expect(hook.result.current.calculation.shares.get("user-a")).toBe(
            650n,
        );
        expect(hook.result.current.calculation.shares.get("user-b")).toBe(
            550n,
        );
    });

    it("preserves each mode draft while switching modes", () => {
        const hook = renderDraft({
            mode: "equal",
            participants: members.map(({ userId }) => ({ userId })),
        });

        act(() => hook.result.current.setMode("exact"));
        act(() =>
            hook.result.current.updateEntry("user-a", (entry) => ({
                ...entry,
                amount: "7.00",
            })),
        );
        act(() => hook.result.current.setMode("percentage"));
        act(() => hook.result.current.setMode("exact"));

        expect(hook.result.current.entries[0].amount).toBe("7.00");
    });
});
