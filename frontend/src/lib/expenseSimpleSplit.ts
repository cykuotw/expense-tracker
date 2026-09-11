import { ExpenseAllocation } from "../types/allocation";
import { GroupMember } from "../types/group";

export type SimpleSplitChoice =
    | "equal"
    | "you-equal"
    | "you-full"
    | "other-equal"
    | "other-full"
    | "unequal";

function selectedParticipantIDs(allocation: ExpenseAllocation): Set<string> {
    return new Set(allocation.participants.map(({ userId }) => userId));
}

function exactlySelected(
    selected: Set<string>,
    expected: string[]
): boolean {
    return (
        selected.size === expected.length &&
        expected.every((userId) => selected.has(userId))
    );
}

export function simpleSplitChoice(
    members: GroupMember[],
    currentUserId: string,
    payerUserId: string,
    allocation: ExpenseAllocation
): SimpleSplitChoice {
    if (allocation.mode !== "equal" || members.length === 0) {
        return "unequal";
    }

    const selected = selectedParticipantIDs(allocation);
    if (members.length === 1) {
        return exactlySelected(selected, [members[0].userId])
            ? "equal"
            : "unequal";
    }

    if (members.length === 2) {
        const otherUserId = members.find(
            ({ userId }) => userId !== currentUserId
        )?.userId;
        if (!otherUserId || !members.some(({ userId }) => userId === currentUserId)) {
            return "unequal";
        }
        if (exactlySelected(selected, [otherUserId, currentUserId])) {
            if (payerUserId === currentUserId) return "you-equal";
            if (payerUserId === otherUserId) return "other-equal";
        }
        if (
            payerUserId === currentUserId &&
            exactlySelected(selected, [currentUserId])
        ) {
            return "you-full";
        }
        if (
            payerUserId === otherUserId &&
            exactlySelected(selected, [otherUserId])
        ) {
            return "other-full";
        }
        return "unequal";
    }

    return exactlySelected(
        selected,
        members.map(({ userId }) => userId)
    )
        ? "equal"
        : "unequal";
}

export function simpleSplitSelection(
    choice: Exclude<SimpleSplitChoice, "unequal">,
    members: GroupMember[],
    currentUserId: string,
    currentPayerUserId: string
): { payerUserId: string; allocation: ExpenseAllocation } | null {
    if (members.length === 0) return null;

    if (!members.some(({ userId }) => userId === currentUserId)) return null;
    const otherUserId = members.find(
        ({ userId }) => userId !== currentUserId
    )?.userId;
    const allParticipants = members.map(({ userId }) => ({ userId }));

    switch (choice) {
        case "you-equal":
            return {
                payerUserId: currentUserId,
                allocation: { mode: "equal", participants: allParticipants },
            };
        case "you-full":
            return {
                payerUserId: currentUserId,
                allocation: {
                    mode: "equal",
                    participants: [{ userId: currentUserId }],
                },
            };
        case "other-equal":
            if (!otherUserId) return null;
            return {
                payerUserId: otherUserId,
                allocation: { mode: "equal", participants: allParticipants },
            };
        case "other-full":
            if (!otherUserId) return null;
            return {
                payerUserId: otherUserId,
                allocation: {
                    mode: "equal",
                    participants: [{ userId: otherUserId }],
                },
            };
        case "equal":
            return {
                payerUserId: currentPayerUserId || currentUserId,
                allocation: { mode: "equal", participants: allParticipants },
            };
    }
}
