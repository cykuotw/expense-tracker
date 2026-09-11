import { useMemo, useState } from "react";

import {
    basisPointsToPercentage,
    calculateExpenseAllocation,
    percentageInputToBasisPoints,
} from "../../lib/expenseAllocation";
import { unitsToDecimal } from "../../lib/money";
import {
    ExpenseAllocation,
    ExpenseAllocationMode,
} from "../../types/allocation";
import { GroupMember } from "../../types/group";

export interface SplitDraftEntry {
    userId: string;
    selected: boolean;
    amount: string;
    percentage: string;
}

type ModeDrafts = Record<ExpenseAllocationMode, SplitDraftEntry[]>;

interface UseExpenseSplitDraftOptions {
    allocation: ExpenseAllocation;
    amountDigits: number | null;
    currency: string;
    members: GroupMember[];
    total: string;
}

function initialModeDrafts({
    allocation,
    amountDigits,
    currency,
    members,
    total,
}: UseExpenseSplitDraftOptions): ModeDrafts {
    const selected = new Set(
        allocation.participants.map(({ userId }) => userId),
    );
    const selectedIDs = members
        .map(({ userId }) => userId)
        .filter((userId) => selected.has(userId));
    const defaultIDs = selectedIDs.length
        ? selectedIDs
        : members.map(({ userId }) => userId);
    const defaultSelected = new Set(defaultIDs);
    const calculation = calculateExpenseAllocation(
        total,
        amountDigits,
        allocation,
        currency,
    );
    const orderedIDs = [...defaultIDs].sort((left, right) =>
        left.localeCompare(right),
    );
    const divisor = Math.max(orderedIDs.length, 1);
    const percentageBase = Math.floor(10_000 / divisor);
    const percentageRemainder = 10_000 % divisor;
    const defaultPercentages = new Map(
        orderedIDs.map((userId, index) => [
            userId,
            percentageBase + (index < percentageRemainder ? 1 : 0),
        ]),
    );
    const stored = new Map(
        allocation.participants.map((participant) => [
            participant.userId,
            participant,
        ]),
    );

    const entriesFor = (mode: ExpenseAllocationMode): SplitDraftEntry[] =>
        members.map(({ userId }) => {
            const participant = stored.get(userId);
            const isActiveMode = allocation.mode === mode;
            const share = calculation.shares.get(userId);
            return {
                userId,
                selected: isActiveMode
                    ? selected.has(userId)
                    : defaultSelected.has(userId),
                amount:
                    isActiveMode && participant?.amount !== undefined
                        ? participant.amount
                        : mode === "exact" &&
                            share !== undefined &&
                            amountDigits !== null
                          ? unitsToDecimal(share, amountDigits)
                          : "0",
                percentage:
                    isActiveMode &&
                    participant?.percentageBasisPoints !== undefined
                        ? basisPointsToPercentage(
                              participant.percentageBasisPoints,
                          )
                        : basisPointsToPercentage(
                              defaultPercentages.get(userId) ?? 0,
                          ),
            };
        });

    return {
        equal: entriesFor("equal"),
        exact: entriesFor("exact"),
        percentage: entriesFor("percentage"),
        adjustment: entriesFor("adjustment"),
    };
}

function allocationFromDraft(
    mode: ExpenseAllocationMode,
    entries: SplitDraftEntry[],
): ExpenseAllocation {
    return {
        mode,
        participants: entries
            .filter(({ selected }) => selected)
            .map(({ userId, amount, percentage }) => {
                if (mode === "exact" || mode === "adjustment") {
                    return { userId, amount: amount || "0" };
                }
                if (mode === "percentage") {
                    return {
                        userId,
                        percentageBasisPoints:
                            percentageInputToBasisPoints(percentage) ?? -1,
                    };
                }
                return { userId };
            })
            .sort((left, right) => left.userId.localeCompare(right.userId)),
    };
}

export function useExpenseSplitDraft(options: UseExpenseSplitDraftOptions) {
    const [mode, setMode] = useState<ExpenseAllocationMode>(
        options.allocation.mode,
    );
    const [drafts, setDrafts] = useState<ModeDrafts>(() =>
        initialModeDrafts(options),
    );
    const [touchedFields, setTouchedFields] = useState<Set<string>>(
        () => new Set(),
    );
    const [failedSaveCount, setFailedSaveCount] = useState(0);
    const entries = drafts[mode];
    const draftAllocation = useMemo(
        () => allocationFromDraft(mode, entries),
        [entries, mode],
    );
    const calculation = useMemo(
        () =>
            calculateExpenseAllocation(
                options.total,
                options.amountDigits,
                draftAllocation,
                options.currency,
            ),
        [
            draftAllocation,
            options.amountDigits,
            options.currency,
            options.total,
        ],
    );

    const updateEntry = (
        userId: string,
        update: (entry: SplitDraftEntry) => SplitDraftEntry,
    ) => {
        setDrafts((current) => ({
            ...current,
            [mode]: current[mode].map((entry) =>
                entry.userId === userId ? update(entry) : entry,
            ),
        }));
    };
    const touchField = (userId: string) => {
        setTouchedFields((current) =>
            new Set(current).add(`${mode}:${userId}`),
        );
    };
    const markInvalidSave = () => {
        setTouchedFields(
            new Set(
                entries
                    .filter(({ selected }) => selected)
                    .map(({ userId }) => `${mode}:${userId}`),
            ),
        );
        setFailedSaveCount((count) => count + 1);
    };

    return {
        calculation,
        draftAllocation,
        entries,
        failedSaveCount,
        markInvalidSave,
        mode,
        setMode,
        touchedFields,
        touchField,
        updateEntry,
    };
}
