import { mdiAccountMultipleOutline, mdiChevronRight } from "@mdi/js";
import Icon from "@mdi/react";
import { Link } from "react-router-dom";
import { AllocationCalculation } from "../../lib/expenseAllocation";
import { unitsToDecimal } from "../../lib/money";
import { ExpenseAllocation } from "../../types/allocation";
import { GroupMember } from "../../types/group";
import {
    Card,
    CardAction,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "../ui/card";

const MODE_LABELS: Record<ExpenseAllocation["mode"], string> = {
    equal: "Equally",
    exact: "Exact amounts",
    percentage: "Percentage",
    adjustment: "Equal + adjustments",
};

interface ExpenseSplitSummaryProps {
    allocation: ExpenseAllocation;
    amountDigits: number | null;
    calculation: AllocationCalculation;
    currency: string;
    groupMembers: GroupMember[];
    to: string;
}

export default function ExpenseSplitSummary({
    allocation,
    amountDigits,
    calculation,
    currency,
    groupMembers,
    to,
}: ExpenseSplitSummaryProps) {
    const usernames = new Map(
        groupMembers.map(({ userId, username }) => [userId, username])
    );

    return (
        <Card className="mt-4 md:mt-6">
            <CardHeader>
                <CardTitle>Split expense</CardTitle>
                <CardDescription>
                    {MODE_LABELS[allocation.mode]} · {allocation.participants.length}{" "}
                    {allocation.participants.length === 1 ? "person" : "people"}
                </CardDescription>
                <CardAction>
                    <Link
                        className="ui-button ui-button-outline min-h-11 gap-2 px-3"
                        to={to}
                    >
                        Edit split
                        <Icon path={mdiChevronRight} size={0.8} aria-hidden="true" />
                    </Link>
                </CardAction>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
                {allocation.participants.map(({ userId }) => {
                    const share = calculation.shares.get(userId);
                    return (
                        <div
                            className="flex min-w-0 items-center justify-between gap-3 text-sm"
                            key={userId}
                        >
                            <span className="flex min-w-0 items-center gap-2 text-muted-foreground">
                                <Icon
                                    path={mdiAccountMultipleOutline}
                                    size={0.75}
                                    aria-hidden="true"
                                />
                                <span className="truncate">
                                    {usernames.get(userId) ?? "Unknown member"}
                                </span>
                            </span>
                            <span className="shrink-0 font-medium tabular-nums">
                                {share !== undefined && amountDigits !== null
                                    ? unitsToDecimal(share, amountDigits)
                                    : "—"}{" "}
                                {currency}
                            </span>
                        </div>
                    );
                })}
                {!calculation.valid ? (
                    <p className="text-sm text-destructive" role="alert">
                        {calculation.message}
                    </p>
                ) : null}
            </CardContent>
        </Card>
    );
}
