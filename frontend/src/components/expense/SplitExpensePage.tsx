import { FormEvent, useEffect, useRef } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { toast } from "react-hot-toast";
import { mdiArrowLeft, mdiCheck } from "@mdi/js";
import Icon from "@mdi/react";

import { ExpenseAllocation } from "../../types/allocation";
import { GroupMember } from "../../types/group";
import MobilePageHeader from "../MobilePageHeader";
import { Button } from "../ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "../ui/card";
import ExpenseAllocationModeSelector from "./ExpenseAllocationModeSelector";
import ExpenseParticipantAllocationTable from "./ExpenseParticipantAllocationTable";
import { useExpenseSplitDraft } from "./useExpenseSplitDraft";

const DIRECT_ROUTE_TOAST_ID = "expense-split-direct-route-recovery";

interface SplitExpensePageProps {
    allocation: ExpenseAllocation;
    amountDigits: number | null;
    currency: string;
    groupMembers: GroupMember[];
    mainFormVisited: boolean;
    onSave: (allocation: ExpenseAllocation) => void;
    returnTo: string;
    total: string;
}

export default function SplitExpensePage({
    allocation,
    amountDigits,
    currency,
    groupMembers,
    mainFormVisited,
    onSave,
    returnTo,
    total,
}: SplitExpensePageProps) {
    const navigate = useNavigate();
    const errorSummaryRef = useRef<HTMLDivElement>(null);
    const {
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
    } = useExpenseSplitDraft({
        allocation,
        amountDigits,
        currency,
        members: groupMembers,
        total,
    });

    useEffect(() => {
        if (!mainFormVisited) {
            toast.error("Open the expense form before configuring its split.", {
                id: DIRECT_ROUTE_TOAST_ID,
            });
        }
    }, [mainFormVisited]);

    useEffect(() => {
        if (failedSaveCount > 0 && !calculation.valid) {
            errorSummaryRef.current?.focus();
        }
    }, [calculation.valid, failedSaveCount]);

    if (!mainFormVisited) {
        return <Navigate replace to={returnTo} />;
    }

    const handleSubmit = (event: FormEvent) => {
        event.preventDefault();
        if (!calculation.valid) {
            markInvalidSave();
            return;
        }
        onSave(draftAllocation);
        navigate(returnTo);
    };

    return (
        <div className="page-shell expense-form-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title="Split expense"
                    backTo={returnTo}
                    backLabel="Cancel split changes"
                />

                <div className="page-header desktop-page-header expense-form-header">
                    <div className="page-header__copy expense-form-header__copy">
                        <div className="page-eyebrow">Expense</div>
                        <h1 className="page-title">Split expense</h1>
                        <p className="page-copy">
                            Choose who owes this {total || "0"} {currency}{" "}
                            expense.
                        </p>
                    </div>
                    <Button
                        className="min-h-11"
                        type="button"
                        variant="ghost"
                        onClick={() => navigate(returnTo)}
                    >
                        <Icon
                            path={mdiArrowLeft}
                            data-icon="inline-start"
                            aria-hidden="true"
                        />
                        Cancel
                    </Button>
                </div>

                <form
                    className="flex flex-col gap-5 pb-28 md:pb-8"
                    noValidate
                    onSubmit={handleSubmit}
                >
                    <div
                        data-slot="split-editor"
                        className="overflow-hidden rounded-xl bg-card text-card-foreground ring-1 ring-foreground/10 md:contents"
                    >
                        <Card className="rounded-b-none bg-transparent pb-0 ring-0 md:rounded-xl md:bg-card md:pb-(--card-spacing) md:ring-1">
                            <CardHeader>
                                <CardTitle>How should it be split?</CardTitle>
                                <CardDescription>
                                    Your selection controls the fields shown
                                    below.
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="min-w-0">
                                <ExpenseAllocationModeSelector
                                    mode={mode}
                                    onModeChange={setMode}
                                />
                            </CardContent>
                        </Card>

                        <ExpenseParticipantAllocationTable
                            amountDigits={amountDigits}
                            calculation={calculation}
                            currency={currency}
                            entries={entries}
                            errorSummaryRef={errorSummaryRef}
                            members={groupMembers}
                            mode={mode}
                            onTouchField={touchField}
                            onUpdateEntry={updateEntry}
                            touchedFields={touchedFields}
                        />
                    </div>

                    <div className="expense-split-actions">
                        <Button
                            className="min-h-12 w-full md:w-auto md:min-w-36"
                            type="submit"
                        >
                            <Icon
                                path={mdiCheck}
                                data-icon="inline-start"
                                aria-hidden="true"
                            />
                            Save split
                        </Button>
                    </div>
                </form>
            </div>
        </div>
    );
}
