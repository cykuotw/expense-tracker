import { RefObject, useMemo } from "react";

import { AllocationCalculation, percentageInputToBasisPoints } from "../../lib/expenseAllocation";
import {
    decimalToUnits,
    moneyInputPlaceholder,
    moneyInputStep,
    unitsToDecimal,
} from "../../lib/money";
import { cn } from "../../lib/utils";
import { ExpenseAllocationMode } from "../../types/allocation";
import { GroupMember } from "../../types/group";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "../ui/card";
import { Checkbox } from "../ui/checkbox";
import { SplitDraftEntry } from "./useExpenseSplitDraft";

interface ExpenseParticipantAllocationTableProps {
    amountDigits: number | null;
    calculation: AllocationCalculation;
    currency: string;
    entries: SplitDraftEntry[];
    errorSummaryRef: RefObject<HTMLDivElement | null>;
    members: GroupMember[];
    mode: ExpenseAllocationMode;
    onTouchField: (userId: string) => void;
    onUpdateEntry: (
        userId: string,
        update: (entry: SplitDraftEntry) => SplitDraftEntry,
    ) => void;
    touchedFields: Set<string>;
}

function formattedShare(
    share: bigint,
    amountDigits: number | null,
    currency: string,
) {
    return `${amountDigits === null ? "—" : unitsToDecimal(share, amountDigits)} ${currency}`;
}

export default function ExpenseParticipantAllocationTable({
    amountDigits,
    calculation,
    currency,
    entries,
    errorSummaryRef,
    members,
    mode,
    onTouchField,
    onUpdateEntry,
    touchedFields,
}: ExpenseParticipantAllocationTableProps) {
    const membersByID = useMemo(
        () => new Map(members.map((member) => [member.userId, member])),
        [members],
    );
    const showAmount = mode === "exact" || mode === "adjustment";
    const showPercentage = mode === "percentage";
    const hasValueInput = showAmount || showPercentage;
    const showDerivedShare = mode !== "exact";
    const valueColumnLabel = showAmount
        ? `${mode === "exact" ? "Amount" : "Extra"} (${currency})`
        : showPercentage
          ? "Percentage (%)"
          : "Final share";
    const participantGridColumns = hasValueInput
        ? showDerivedShare
            ? "grid-cols-[minmax(0,1fr)_minmax(7.5rem,0.85fr)] sm:grid-cols-[minmax(12rem,1fr)_minmax(12rem,0.8fr)_minmax(8rem,0.55fr)]"
            : "grid-cols-[minmax(0,1fr)_minmax(7.5rem,0.85fr)] sm:grid-cols-[minmax(12rem,1fr)_minmax(12rem,0.8fr)]"
        : "grid-cols-[minmax(0,1fr)_minmax(7.5rem,auto)]";

    const fieldError = (entry: SplitDraftEntry) => {
        if (!entry.selected) return "";
        if (showAmount) {
            const units =
                amountDigits === null
                    ? null
                    : decimalToUnits(entry.amount, amountDigits);
            return units === null || units < 0n
                ? `Enter a non-negative ${currency} amount with the supported precision.`
                : "";
        }
        if (
            showPercentage &&
            percentageInputToBasisPoints(entry.percentage) === null
        ) {
            return "Enter a percentage from 0 to 100 with at most two decimal places.";
        }
        return "";
    };

    const firstInvalidEntry = entries.find(
        (entry) => fieldError(entry).length > 0,
    );
    const firstReviewEntry =
        firstInvalidEntry ?? entries.find(({ selected }) => selected);
    const summaryTone = calculation.valid
        ? "border-primary/25 bg-primary/5 text-foreground"
        : "border-destructive/35 bg-destructive/5 text-destructive";

    return (
        <Card className="rounded-t-none bg-transparent pt-2 ring-0 md:rounded-xl md:bg-card md:pt-(--card-spacing) md:ring-1">
            <CardHeader>
                <CardTitle>Participants</CardTitle>
                <CardDescription>
                    Select at least one person. A zero value still keeps a
                    person selected.
                </CardDescription>
            </CardHeader>
            <CardContent id="split-participants">
                <fieldset className="overflow-hidden rounded-xl border border-border">
                    <legend className="sr-only">Participant allocations</legend>
                    <div
                        data-slot="split-participant-header"
                        className={cn(
                            "grid items-center gap-x-3 bg-muted/60 px-3 py-2 text-xs font-medium text-muted-foreground",
                            participantGridColumns,
                        )}
                    >
                        <span>Person</span>
                        <span>{valueColumnLabel}</span>
                        {hasValueInput && showDerivedShare ? (
                            <span className="hidden text-right sm:block">
                                Final share
                            </span>
                        ) : null}
                    </div>
                    <div className="divide-y divide-border">
                        {entries.map((entry) => {
                            const member = membersByID.get(entry.userId);
                            const share = calculation.shares.get(entry.userId);
                            const fieldKey = `${mode}:${entry.userId}`;
                            const inputID = `split-${mode}-${entry.userId}`;
                            const error = fieldError(entry);
                            const showFieldError =
                                touchedFields.has(fieldKey) && error.length > 0;
                            const inputLabel = showPercentage
                                ? `${member?.username ?? "Participant"} percentage (%)`
                                : `${member?.username ?? "Participant"} ${mode === "exact" ? "amount" : "extra"} (${currency})`;
                            return (
                                <div
                                    data-slot="split-participant"
                                    className={cn(
                                        "grid min-w-0 items-center gap-x-3 gap-y-1 px-3 py-2",
                                        participantGridColumns,
                                    )}
                                    key={entry.userId}
                                >
                                    <div className="min-w-0">
                                        <label className="flex min-h-11 cursor-pointer items-center gap-3">
                                            <Checkbox
                                                checked={entry.selected}
                                                onCheckedChange={(checked) =>
                                                    onUpdateEntry(
                                                        entry.userId,
                                                        (current) => ({
                                                            ...current,
                                                            selected:
                                                                checked === true,
                                                        }),
                                                    )
                                                }
                                                aria-label={`Include ${member?.username ?? "participant"}`}
                                            />
                                            <span className="min-w-0 font-medium break-words">
                                                {member?.username ??
                                                    "Unknown member"}
                                            </span>
                                        </label>
                                        {entry.selected &&
                                        share !== undefined &&
                                        hasValueInput &&
                                        showDerivedShare ? (
                                            <output className="block pl-7 text-xs text-muted-foreground sm:hidden">
                                                Final share: {formattedShare(share, amountDigits, currency)}
                                            </output>
                                        ) : null}
                                    </div>

                                    {entry.selected && hasValueInput ? (
                                        <label
                                            className="flex min-w-0 flex-col gap-1"
                                            htmlFor={inputID}
                                        >
                                            <span className="sr-only">
                                                {inputLabel}
                                            </span>
                                            <input
                                                id={inputID}
                                                className="ui-input-shell min-h-11 min-w-0 bg-background"
                                                type="number"
                                                inputMode="decimal"
                                                min="0"
                                                max={showPercentage ? "100" : undefined}
                                                step={
                                                    showPercentage
                                                        ? "0.01"
                                                        : amountDigits === null
                                                          ? undefined
                                                          : moneyInputStep(
                                                                amountDigits,
                                                            )
                                                }
                                                placeholder={
                                                    showPercentage
                                                        ? undefined
                                                        : amountDigits === null
                                                          ? "Unavailable"
                                                          : moneyInputPlaceholder(
                                                                amountDigits,
                                                            )
                                                }
                                                value={
                                                    showPercentage
                                                        ? entry.percentage
                                                        : entry.amount
                                                }
                                                aria-invalid={showFieldError}
                                                aria-describedby={
                                                    showFieldError
                                                        ? `${inputID}-error`
                                                        : undefined
                                                }
                                                onChange={(event) =>
                                                    onUpdateEntry(
                                                        entry.userId,
                                                        (current) => ({
                                                            ...current,
                                                            [showPercentage
                                                                ? "percentage"
                                                                : "amount"]:
                                                                event.target
                                                                    .value,
                                                        }),
                                                    )
                                                }
                                                onBlur={() =>
                                                    onTouchField(entry.userId)
                                                }
                                            />
                                            {showFieldError ? (
                                                <span
                                                    id={`${inputID}-error`}
                                                    className="text-xs text-destructive"
                                                >
                                                    {error}
                                                </span>
                                            ) : null}
                                        </label>
                                    ) : null}

                                    {entry.selected &&
                                    share !== undefined &&
                                    !hasValueInput ? (
                                        <output className="text-right text-sm tabular-nums text-muted-foreground">
                                            {formattedShare(share, amountDigits, currency)}
                                        </output>
                                    ) : null}

                                    {entry.selected &&
                                    share !== undefined &&
                                    hasValueInput &&
                                    showDerivedShare ? (
                                        <output className="hidden text-right text-sm tabular-nums text-muted-foreground sm:block">
                                            Final share: {formattedShare(share, amountDigits, currency)}
                                        </output>
                                    ) : null}
                                </div>
                            );
                        })}
                    </div>
                </fieldset>
            </CardContent>
            <CardFooter>
                <div
                    ref={errorSummaryRef}
                    className={`w-full rounded-lg border px-4 py-3 text-sm ${summaryTone}`}
                    role={calculation.valid ? "status" : "alert"}
                    tabIndex={calculation.valid ? undefined : -1}
                >
                    {calculation.message}
                    {!calculation.valid ? (
                        <a
                            className="mt-1 block font-medium underline underline-offset-2"
                            href={
                                firstReviewEntry && hasValueInput
                                    ? `#split-${mode}-${firstReviewEntry.userId}`
                                    : "#split-participants"
                            }
                        >
                            Review split fields
                        </a>
                    ) : null}
                </div>
            </CardFooter>
        </Card>
    );
}
