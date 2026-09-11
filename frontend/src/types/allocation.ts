export type ExpenseAllocationMode =
    | "equal"
    | "exact"
    | "percentage"
    | "adjustment";

export interface ExpenseAllocationParticipant {
    userId: string;
    amount?: string;
    percentageBasisPoints?: number;
}

export interface ExpenseAllocation {
    mode: ExpenseAllocationMode;
    participants: ExpenseAllocationParticipant[];
}
