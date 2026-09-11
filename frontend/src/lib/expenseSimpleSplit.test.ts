import { describe, expect, it } from "vitest";

import { ExpenseAllocation } from "../types/allocation";
import { GroupMember } from "../types/group";
import { simpleSplitChoice, simpleSplitSelection } from "./expenseSimpleSplit";

const twoMembers: GroupMember[] = [
    { userId: "other", username: "Alice" },
    { userId: "current", username: "You" },
];
const threeMembers: GroupMember[] = [
    { userId: "alice", username: "Alice" },
    { userId: "bob", username: "Bob" },
    { userId: "current", username: "You" },
];

describe("expenseSimpleSplit", () => {
    it.each([
        ["you-equal", "current", ["other", "current"]],
        ["you-full", "current", ["other"]],
        ["other-equal", "other", ["other", "current"]],
        ["other-full", "other", ["current"]],
    ] as const)(
        "maps the %s two-person shortcut to its canonical allocation",
        (choice, expectedPayer, expectedParticipants) => {
            const selection = simpleSplitSelection(
                choice,
                twoMembers,
                "current",
                ""
            );

            expect(selection).toEqual({
                payerUserId: expectedPayer,
                allocation: {
                    mode: "equal",
                    participants: expectedParticipants.map((userId) => ({
                        userId,
                    })),
                },
            });
            expect(
                simpleSplitChoice(
                    twoMembers,
                    "current",
                    selection?.payerUserId ?? "",
                    selection?.allocation ?? { mode: "equal", participants: [] }
                )
            ).toBe(choice);
        }
    );

    it("uses the current payer when a larger group is split equally", () => {
        const selection = simpleSplitSelection(
            "equal",
            threeMembers,
            "current",
            "bob"
        );

        expect(selection?.payerUserId).toBe("bob");
        expect(selection?.allocation.participants).toEqual(
            threeMembers.map(({ userId }) => ({ userId }))
        );
        expect(
            simpleSplitChoice(
                threeMembers,
                "current",
                "bob",
                selection?.allocation as ExpenseAllocation
            )
        ).toBe("equal");
    });

    it("keeps every advanced allocation behind the Unequally choice", () => {
        expect(
            simpleSplitChoice(twoMembers, "current", "current", {
                mode: "percentage",
                participants: [
                    { userId: "other", percentageBasisPoints: 6000 },
                    { userId: "current", percentageBasisPoints: 4000 },
                ],
            })
        ).toBe("unequal");
        expect(
            simpleSplitChoice(threeMembers, "current", "current", {
                mode: "equal",
                participants: [{ userId: "current" }],
            })
        ).toBe("unequal");
    });

    it("identifies the current user by ID instead of member order", () => {
        const reordered = [twoMembers[1], twoMembers[0]];
        const selection = simpleSplitSelection(
            "you-full",
            reordered,
            "current",
            "other"
        );

        expect(selection).toEqual({
            payerUserId: "current",
            allocation: {
                mode: "equal",
                participants: [{ userId: "other" }],
            },
        });
        expect(
            simpleSplitChoice(
                reordered,
                "current",
                "current",
                selection?.allocation as ExpenseAllocation
            )
        ).toBe("you-full");
    });
});
