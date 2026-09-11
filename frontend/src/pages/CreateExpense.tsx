import { useEffect } from "react";
import { Link, Route, Routes } from "react-router-dom";

import Icon from "@mdi/react";
import {
    mdiCamera,
    mdiCheckBold,
    mdiSubdirectoryArrowLeft,
} from "@mdi/js";

import { CreateExpenseProvider } from "../contexts/CreateExpenseContext";
import { useCreateExpense } from "../hooks/CreateExpenseContextHooks";
import { ExpenseTypePicker } from "../components/expense/ExpenseTypePicker";
import { ExpenseFormPicker } from "../components/expense/ExpenseFormPicker";
import MobilePageHeader from "../components/MobilePageHeader";
import { getGroupTypePresentation } from "../lib/groupTypePresentation";
import { ExpenseDateInput } from "../components/expense/ExpenseDateInput";
import { moneyInputPlaceholder, moneyInputStep } from "../lib/money";
import SplitExpensePage from "../components/expense/SplitExpensePage";
import ExpenseSimpleSplitControl from "../components/expense/ExpenseSimpleSplitControl";

const CreateExpenseContent = () => {
    const {
        groupId,
        selectedGroupId,
        setSelectedGroupId,
        selectedExpenseTypeId,
        setSelectedExpenseTypeId,
        totalInput,
        setTotalInput,
        description,
        setDescription,
        occurredOn,
        setOccurredOn,
        currency,
        amountDigits,
        payer,
        setPayer,
        allocation,
        setAllocation,
        allocationCalculation,
        indicatorShow,
        dataOk,
        groupList,
        expenseTypes,
        groupMembers,
        currentUserId,
        groupMembersLoadStatus,
        reloadGroupMembers,
        handleCreateExpense,
        markMainFormVisited,
    } = useCreateExpense();

    useEffect(() => {
        markMainFormVisited();
    }, [markMainFormVisited]);

    const selectGroup = (value: string) => {
        setSelectedGroupId(value);
    };

    return (
        <div className="page-shell expense-form-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title="Add expense"
                    backTo={groupId ? `/group/${groupId}` : "/"}
                    backLabel="Back to group"
                    action={
                        indicatorShow ? (
                            <span className="ui-spinner ui-spinner-sm" role="status" aria-label="Saving expense" />
                        ) : (
                            <button
                                type="submit"
                                form="create-expense-form"
                                className="ui-button ui-button-primary min-h-12 min-w-12 px-3"
                                aria-label="Save expense"
                                disabled={!dataOk}
                            >
                                <Icon path={mdiCheckBold} size={1} />
                            </button>
                        )
                    }
                />
                <div className="page-header desktop-page-header expense-form-header">
                    <div className="page-header__copy expense-form-header__copy">
                        <div className="page-eyebrow">Expense</div>
                        <h1 className="page-title">Add expense</h1>
                        <p className="page-copy">
                            Track what was paid and split it across the group.
                        </p>
                    </div>
                    <div className="page-actions hidden w-full md:flex md:w-auto">
                        <Link
                            className="ui-button ui-button-ghost w-full sm:w-auto"
                            to={`/group/${groupId}`}
                        >
                            <Icon
                                path={mdiSubdirectoryArrowLeft}
                                size={1}
                            />
                            Back to Group
                        </Link>
                    </div>
                </div>

                <form
                    id="create-expense-form"
                    className="panel-card expense-form-panel rounded-[2rem] p-4 sm:p-6 md:p-8"
                    onSubmit={handleCreateExpense}
                >
                        <div className="grid grid-cols-2 gap-3 md:gap-5">
                            <div className="col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Group
                                </label>
                                <div className="mt-2">
                                    <ExpenseFormPicker
                                        label="Group"
                                        emptyLabel="Choose a group"
                                        value={selectedGroupId ?? ""}
                                        onChange={selectGroup}
                                        options={groupList.map((group) => {
                                            const groupType = getGroupTypePresentation(
                                                group.groupType
                                            );
                                            return {
                                                value: group.id,
                                                label: group.groupName,
                                                description: group.description,
                                                icon: groupType.icon,
                                                iconClassName: groupType.iconClassName,
                                            };
                                        })}
                                    />
                                </div>
                            </div>

                            <div className="col-span-2">
                                <label
                                    htmlFor="occurredOn"
                                    className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60"
                                >
                                    Expense date
                                </label>
                                <ExpenseDateInput
                                    id="occurredOn"
                                    name="occurredOn"
                                    className="mt-2"
                                    value={occurredOn}
                                    onChange={(event) => setOccurredOn(event.target.value)}
                                    required
                                />
                            </div>

                            <div className="col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Expense type
                                </label>
                                <div className="mt-2">
                                    <ExpenseTypePicker
                                        expenseTypes={expenseTypes}
                                        value={selectedExpenseTypeId}
                                        onChange={setSelectedExpenseTypeId}
                                    />
                                </div>
                            </div>

                            <div className="col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Description
                                </label>
                                <label className="expense-form-input-shell mt-2 flex w-full items-center rounded-2xl border border-border bg-background px-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)] transition hover:border-primary/60 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary">
                                    <input
                                        type="text"
                                        name="description"
                                        className="min-w-0 grow border-0 bg-transparent outline-none"
                                        placeholder="Description"
                                        value={description}
                                        onChange={(e) =>
                                            setDescription(e.target.value)
                                        }
                                        required
                                    />
                                </label>
                            </div>

                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Currency
                                </label>
                                <output className="ui-input-shell mt-2 flex min-h-12 items-center bg-background px-4 text-foreground/70">
                                    {currency || "Select a group"}
                                </output>
                            </div>

                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Amount
                                </label>
                                <label className="expense-form-input-shell mt-2 flex w-full items-center rounded-2xl border border-border bg-background px-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)] transition hover:border-primary/60 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary">
                                    <input
                                        type="number"
                                        name="total"
                                        className="min-w-0 grow border-0 bg-transparent outline-none"
                                        step={amountDigits === null ? undefined : moneyInputStep(amountDigits)}
                                        placeholder={amountDigits === null ? "Select a group" : moneyInputPlaceholder(amountDigits)}
                                        value={totalInput}
                                        onChange={(e) =>
                                            setTotalInput(e.target.value)
                                        }
                                        required
                                        min={0}
                                        disabled={amountDigits === null}
                                    />
                                </label>
                            </div>
                        </div>

                        {/* RECEIPT UPLOAD BUTTON */}
                        <div className="hidden">
                            <label
                                style={{ display: "inline-block" }}
                                className="w-2/3 h-12 border border-gray-400 rounded-full bg-background hover:bg-border"
                            >
                                <input
                                    type="file"
                                    style={{ display: "none" }}
                                />
                                <div className="flex flex-row items-center justify-center h-full space-x-3">
                                    <Icon path={mdiCamera} size={1} />
                                    <p>Upload Receipt</p>
                                </div>
                            </label>
                        </div>

                        <div className="mt-4 md:mt-6">
                            {groupMembersLoadStatus === "loading" ? (
                                <p
                                    role="status"
                                    className="flex min-h-14 items-center rounded-2xl border border-border bg-muted/40 px-4 text-sm text-foreground/60 md:min-h-20"
                                >
                                    Loading split options…
                                </p>
                            ) : groupMembersLoadStatus === "error" ? (
                                <div className="flex min-h-14 items-center justify-between gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-2 md:min-h-20">
                                    <p className="text-sm text-destructive">
                                        Split options could not be loaded.
                                    </p>
                                    <button
                                        type="button"
                                        className="ui-button ui-button-ghost min-h-11 shrink-0 px-3"
                                        onClick={reloadGroupMembers}
                                    >
                                        Try again
                                    </button>
                                </div>
                            ) : groupMembers.length === 0 ? (
                                <p
                                    role="status"
                                    className="flex min-h-14 items-center rounded-2xl border border-border bg-muted/40 px-4 text-sm text-foreground/60 md:min-h-20"
                                >
                                    Choose a group to see split options.
                                </p>
                            ) : (
                                <>
                                    <ExpenseSimpleSplitControl
                                        allocation={allocation}
                                        amountDigits={amountDigits}
                                        calculation={allocationCalculation}
                                        currency={currency}
                                        groupMembers={groupMembers}
                                        currentUserId={currentUserId}
                                        onAllocationChange={setAllocation}
                                        onPayerChange={setPayer}
                                        payerUserId={payer}
                                        splitTo={`split${selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : ""}`}
                                    />
                                </>
                            )}
                        </div>

                        <div className="mt-5 hidden flex-col gap-3 md:flex md:flex-row md:items-center md:justify-between">
                            <button
                                type="submit"
                                className="ui-button ui-button-primary w-full sm:w-auto"
                                {...(dataOk ? {} : { disabled: true })}
                            >
                                <Icon path={mdiCheckBold} size={1} />
                                Save Expense
                            </button>
                        </div>

                        {indicatorShow && (
                            <div className="hidden justify-center pt-4 md:flex">
                                <span className="ui-spinner ui-spinner-sm"></span>
                            </div>
                        )}
                </form>
            </div>
        </div>
    );
};

const CreateExpense = () => {
    return (
        <CreateExpenseProvider>
            <Routes>
                <Route index element={<CreateExpenseContent />} />
                <Route path="split" element={<CreateExpenseSplit />} />
            </Routes>
        </CreateExpenseProvider>
    );
};

const CreateExpenseSplit = () => {
    const {
        allocation,
        amountDigits,
        currency,
        groupMembers,
        mainFormVisited,
        selectedGroupId,
        setAllocation,
        totalInput,
    } = useCreateExpense();
    const returnTo = `/create_expense${selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : ""}`;

    return (
        <SplitExpensePage
            allocation={allocation}
            amountDigits={amountDigits}
            currency={currency}
            groupMembers={groupMembers}
            mainFormVisited={mainFormVisited}
            onSave={setAllocation}
            returnTo={returnTo}
            total={totalInput}
        />
    );
};

export default CreateExpense;
