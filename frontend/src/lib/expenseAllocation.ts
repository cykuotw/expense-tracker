import {
    ExpenseAllocation,
    ExpenseAllocationParticipant,
} from "../types/allocation";
import { decimalToUnits, splitEqually, unitsToDecimal } from "./money";

export interface AllocationCalculation {
    valid: boolean;
    message: string;
    remainingUnits: bigint;
    shares: Map<string, bigint>;
}

const invalidCalculation = (
    message: string,
    remainingUnits = 0n
): AllocationCalculation => ({
    valid: false,
    message,
    remainingUnits,
    shares: new Map(),
});

export function calculateExpenseAllocation(
    total: string,
    amountDigits: number | null,
    allocation: ExpenseAllocation,
    currency: string
): AllocationCalculation {
    if (amountDigits === null) {
        return invalidCalculation("Currency metadata is unavailable.");
    }
    const totalUnits = decimalToUnits(total, amountDigits);
    if (totalUnits === null || totalUnits <= 0n) {
        return invalidCalculation("Enter a valid expense amount first.");
    }
    if (
        allocation.participants.length === 0 ||
        new Set(allocation.participants.map(({ userId }) => userId)).size !==
            allocation.participants.length
    ) {
        return invalidCalculation("Select at least one participant.", totalUnits);
    }

    const participants = [...allocation.participants].sort((left, right) =>
        left.userId.localeCompare(right.userId)
    );

    switch (allocation.mode) {
        case "equal":
            return equalCalculation(totalUnits, participants);
        case "exact":
            return exactCalculation(
                totalUnits,
                amountDigits,
                participants,
                currency
            );
        case "percentage":
            return percentageCalculation(totalUnits, participants);
        case "adjustment":
            return adjustmentCalculation(
                totalUnits,
                amountDigits,
                participants,
                currency
            );
    }
}

function equalCalculation(
    totalUnits: bigint,
    participants: ExpenseAllocationParticipant[]
): AllocationCalculation {
    const shares = splitEqually(
        totalUnits,
        participants.map(({ userId }) => userId)
    );
    if (shares === null) {
        return invalidCalculation("Select at least one participant.", totalUnits);
    }
    return { valid: true, message: "Split total is complete.", remainingUnits: 0n, shares };
}

function exactCalculation(
    totalUnits: bigint,
    amountDigits: number,
    participants: ExpenseAllocationParticipant[],
    currency: string
): AllocationCalculation {
    const shares = new Map<string, bigint>();
    let allocated = 0n;
    for (const participant of participants) {
        const units = decimalToUnits(participant.amount ?? "", amountDigits);
        if (units === null || units < 0n) {
            return invalidCalculation(
                `Enter a valid ${currency} amount for every selected participant.`,
                totalUnits - allocated
            );
        }
        shares.set(participant.userId, units);
        allocated += units;
    }

    const remainingUnits = totalUnits - allocated;
    if (remainingUnits !== 0n) {
        const difference =
            remainingUnits < 0n ? -remainingUnits : remainingUnits;
        return {
            valid: false,
            message:
                remainingUnits < 0n
                    ? `${unitsToDecimal(difference, amountDigits)} ${currency} over allocated.`
                    : `${unitsToDecimal(difference, amountDigits)} ${currency} left to allocate.`,
            remainingUnits,
            shares,
        };
    }
    return { valid: true, message: "Split total is complete.", remainingUnits, shares };
}

function percentageCalculation(
    totalUnits: bigint,
    participants: ExpenseAllocationParticipant[]
): AllocationCalculation {
    if (participants.some(
        ({ percentageBasisPoints }) =>
            percentageBasisPoints === undefined ||
            !Number.isInteger(percentageBasisPoints) ||
            percentageBasisPoints < 0 ||
            percentageBasisPoints > 10_000
    )) {
        return invalidCalculation(
            "Enter a percentage from 0 to 100 with at most two decimal places for every selected participant."
        );
    }
    const percentageTotal = participants.reduce(
        (sum, participant) => sum + (participant.percentageBasisPoints ?? 0),
        0
    );
    if (percentageTotal !== 10_000) {
        const difference = Math.abs(10_000 - percentageTotal) / 100;
        return invalidCalculation(
            percentageTotal > 10_000
                ? `${difference.toFixed(2)}% over allocated.`
                : `${difference.toFixed(2)}% left to allocate.`
        );
    }

    const shares = new Map<string, bigint>();
    const fractions = participants.map((participant) => {
        const numerator =
            totalUnits * BigInt(participant.percentageBasisPoints ?? 0);
        const units = numerator / 10_000n;
        shares.set(participant.userId, units);
        return {
            userId: participant.userId,
            remainder: numerator % 10_000n,
        };
    });
    const allocated = [...shares.values()].reduce((sum, units) => sum + units, 0n);
    const remainderUnits = totalUnits - allocated;
    fractions.sort((left, right) => {
        if (left.remainder === right.remainder) {
            return left.userId.localeCompare(right.userId);
        }
        return left.remainder > right.remainder ? -1 : 1;
    });
    for (let index = 0; index < Number(remainderUnits); index += 1) {
        const userId = fractions[index].userId;
        shares.set(userId, (shares.get(userId) ?? 0n) + 1n);
    }
    return { valid: true, message: "Percentages total 100%.", remainingUnits: 0n, shares };
}

function adjustmentCalculation(
    totalUnits: bigint,
    amountDigits: number,
    participants: ExpenseAllocationParticipant[],
    currency: string
): AllocationCalculation {
    const adjustments = new Map<string, bigint>();
    let adjustmentTotal = 0n;
    for (const participant of participants) {
        const units = decimalToUnits(participant.amount ?? "0", amountDigits);
        if (units === null || units < 0n) {
            return invalidCalculation(
                `Enter a valid ${currency} adjustment for every selected participant.`
            );
        }
        adjustments.set(participant.userId, units);
        adjustmentTotal += units;
    }
    if (adjustmentTotal > totalUnits) {
        return invalidCalculation("Adjustments cannot exceed the expense total.");
    }

    const base = splitEqually(
        totalUnits - adjustmentTotal,
        participants.map(({ userId }) => userId)
    );
    if (base === null) {
        return invalidCalculation("Select at least one participant.");
    }
    const shares = new Map(
        participants.map(({ userId }) => [
            userId,
            (base.get(userId) ?? 0n) + (adjustments.get(userId) ?? 0n),
        ])
    );
    return { valid: true, message: "Split total is complete.", remainingUnits: 0n, shares };
}

export function percentageInputToBasisPoints(value: string): number | null {
    const match = /^(?:0|[1-9]\d*)(?:\.(\d{0,2}))?$/.exec(value);
    if (match === null) return null;
    const [whole] = value.split(".");
    const fraction = (match[1] ?? "").padEnd(2, "0");
    const basisPoints = Number(whole) * 100 + Number(fraction || "0");
    return Number.isSafeInteger(basisPoints) && basisPoints <= 10_000
        ? basisPoints
        : null;
}

export function basisPointsToPercentage(value: number): string {
    const whole = Math.floor(value / 100);
    const fraction = String(value % 100).padStart(2, "0").replace(/0+$/, "");
    return fraction ? `${whole}.${fraction}` : String(whole);
}
