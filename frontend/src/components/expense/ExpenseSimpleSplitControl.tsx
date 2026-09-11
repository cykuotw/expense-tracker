import { mdiAccountOutline } from "@mdi/js";
import { useNavigate } from "react-router-dom";

import { AllocationCalculation } from "../../lib/expenseAllocation";
import {
    SimpleSplitChoice,
    simpleSplitChoice,
    simpleSplitSelection,
} from "../../lib/expenseSimpleSplit";
import { ExpenseAllocation } from "../../types/allocation";
import { GroupMember } from "../../types/group";
import ExpenseSplitSummary from "./ExpenseSplitSummary";
import {
    ExpenseFormPicker,
    ExpenseFormPickerOption,
} from "./ExpenseFormPicker";

interface ExpenseSimpleSplitControlProps {
    allocation: ExpenseAllocation;
    amountDigits: number | null;
    calculation: AllocationCalculation;
    currency: string;
    currentUserId: string;
    groupMembers: GroupMember[];
    onAllocationChange: (allocation: ExpenseAllocation) => void;
    onPayerChange: (payerUserId: string) => void;
    payerUserId: string;
    splitTo: string;
}

export default function ExpenseSimpleSplitControl({
    allocation,
    amountDigits,
    calculation,
    currency,
    currentUserId,
    groupMembers,
    onAllocationChange,
    onPayerChange,
    payerUserId,
    splitTo,
}: ExpenseSimpleSplitControlProps) {
    const navigate = useNavigate();
    const choice = simpleSplitChoice(
        groupMembers,
        currentUserId,
        payerUserId,
        allocation
    );
    const payerOptions: ExpenseFormPickerOption[] = groupMembers.map(
        (member) => ({
            value: member.userId,
            label:
                member.userId === currentUserId ? "You" : member.username,
            description:
                member.userId === currentUserId
                    ? "Your account"
                    : undefined,
        })
    );
    const otherName =
        groupMembers.find(({ userId }) => userId !== currentUserId)?.username ??
        "Other member";
    const twoPersonOptions: ExpenseFormPickerOption[] = [
        { value: "you-equal", label: "You paid, split equally" },
        { value: "you-full", label: "You are owed the full amount" },
        {
            value: "other-equal",
            label: `${otherName} paid, split equally`,
        },
        {
            value: "other-full",
            label: `${otherName} is owed the full amount`,
        },
        {
            value: "unequal",
            label: "Unequally",
            description: "Use exact amounts, percentages, or adjustments",
        },
    ];

    const selectChoice = (value: string) => {
        const nextChoice = value as SimpleSplitChoice;
        if (nextChoice === "unequal") {
            navigate(splitTo);
            return;
        }
        const selection = simpleSplitSelection(
            nextChoice,
            groupMembers,
            currentUserId,
            payerUserId
        );
        if (!selection) return;
        onPayerChange(selection.payerUserId);
        onAllocationChange(selection.allocation);
    };

    if (groupMembers.length === 1) {
        return (
            <ExpenseFormPicker
                label="Split rule"
                emptyLabel="Split equally"
                value="equal"
                onChange={selectChoice}
                options={[
                    {
                        value: "equal",
                        label: "Split equally",
                        description: "Only you are in this group",
                    },
                ]}
            />
        );
    }

    if (groupMembers.length === 2) {
        return (
            <>
                <ExpenseFormPicker
                    label="Split rule"
                    emptyLabel="Choose a split rule"
                    mobileMenuPlacement="above"
                    value={choice}
                    onChange={selectChoice}
                    options={twoPersonOptions}
                />
                {choice === "unequal" ? (
                    <>
                        <div className="mt-4">
                            <ExpenseFormPicker
                                label="Paid by"
                                emptyLabel="Choose a payer"
                                icon={mdiAccountOutline}
                                value={payerUserId}
                                onChange={onPayerChange}
                                options={payerOptions}
                            />
                        </div>
                        <ExpenseSplitSummary
                            allocation={allocation}
                            amountDigits={amountDigits}
                            calculation={calculation}
                            currency={currency}
                            groupMembers={groupMembers}
                            to={splitTo}
                        />
                    </>
                ) : null}
            </>
        );
    }

    return (
        <>
            <ExpenseFormPicker
                label="Paid by"
                emptyLabel="Choose a payer"
                icon={mdiAccountOutline}
                value={payerUserId}
                onChange={onPayerChange}
                options={payerOptions}
            />
            <div className="mt-4">
                <ExpenseFormPicker
                    label="Split"
                    emptyLabel="Choose a split"
                    mobileMenuPlacement="above"
                    value={choice}
                    onChange={selectChoice}
                    options={[
                        { value: "equal", label: "Equally" },
                        {
                            value: "unequal",
                            label: "Unequally",
                            description:
                                "Use exact amounts, percentages, or adjustments",
                        },
                    ]}
                />
            </div>
            {choice === "unequal" ? (
                <ExpenseSplitSummary
                    allocation={allocation}
                    amountDigits={amountDigits}
                    calculation={calculation}
                    currency={currency}
                    groupMembers={groupMembers}
                    to={splitTo}
                />
            ) : null}
        </>
    );
}
