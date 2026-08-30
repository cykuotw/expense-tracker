import Icon from "@mdi/react";
import { Link } from "react-router-dom";
import {
    mdiAccountGroupOutline,
    mdiAccountOutline,
    mdiCamera,
    mdiCheckBold,
    mdiSubdirectoryArrowLeft,
} from "@mdi/js";

import { Rule } from "../types/splitRule";
import { EditExpenseProvider } from "../contexts/EditExpenseContext";
import { useEditExpense } from "../hooks/EditExpenseContextHooks";
import { ExpenseTypePicker } from "../components/expense/ExpenseTypePicker";
import {
    ExpenseFormPicker,
    ExpenseFormPickerOption,
} from "../components/expense/ExpenseFormPicker";

const currencyOptions: ExpenseFormPickerOption[] = [
    { value: "CAD", label: "CAD" },
    { value: "NTD", label: "NTD" },
    { value: "USD", label: "USD" },
];

const EditExpenseContent = () => {
    const {
        formData,
        setFormData,
        groupList,
        expenseTypes,
        groupMembers,
        indicatorShow,
        dataOk,
        ledgerShareOk,
        ledgerShareMessage,
        handleUpdateExpense,
        handleFormDataChange,
    } = useEditExpense();

    const expenseId = window.location.pathname.split("/")[2] || "";
    const payerOptions: ExpenseFormPickerOption[] = groupMembers.map(
        (member, index) => ({
            value: member.userId,
            label:
                index === groupMembers.length - 1 ? "You" : member.username,
            description:
                index === groupMembers.length - 1
                    ? "Your account"
                    : undefined,
        })
    );

    const handleTwoPersonRuleChange = (value: string) => {
        const splitRule = value as Rule;
        let payerUserId = formData.payerUserId;

        switch (splitRule) {
            case Rule.YouHalf:
            case Rule.YouFull:
                payerUserId = groupMembers.at(-1)?.userId ?? "";
                break;
            case Rule.OtherHalf:
            case Rule.OtherFull:
                payerUserId = groupMembers[0]?.userId ?? "";
                break;
            default:
                break;
        }

        setFormData((current) => ({
            ...current,
            splitRule,
            payerUserId,
        }));
    };

    return (
        <div className="page-shell">
            <div className="page-container max-w-5xl">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Expense</div>
                        <h1 className="page-title">Edit expense</h1>
                        <p className="page-copy">
                            Update details and adjust how this expense is split.
                        </p>
                    </div>
                    <div className="page-actions w-full sm:w-auto">
                        <Link
                            className="ui-button ui-button-ghost w-full sm:w-auto"
                            to={`/expense/${expenseId}`}
                        >
                            <Icon
                                path={mdiSubdirectoryArrowLeft}
                                size={1}
                            />
                            Back to Expense
                        </Link>
                    </div>
                </div>

                <form
                    className="panel-card rounded-[2rem] p-6 md:p-8"
                    onSubmit={handleUpdateExpense}
                >
                        <div className="grid gap-5 md:grid-cols-2">
                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Group
                                </label>
                                <div className="mt-2">
                                    <ExpenseFormPicker
                                        label="Group"
                                        emptyLabel="Choose a group"
                                        icon={mdiAccountGroupOutline}
                                        value={formData.groupId}
                                        onChange={(groupId) =>
                                            setFormData((current) => ({
                                                ...current,
                                                groupId,
                                            }))
                                        }
                                        options={groupList.map((group) => ({
                                            value: group.id,
                                            label: group.groupName,
                                            description: group.description,
                                        }))}
                                    />
                                </div>
                            </div>

                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Expense type
                                </label>
                                <div className="mt-2">
                                    <ExpenseTypePicker
                                        expenseTypes={expenseTypes}
                                        value={formData.expenseType}
                                        onChange={(expenseType) =>
                                            setFormData((current) => ({
                                                ...current,
                                                expenseType,
                                            }))
                                        }
                                    />
                                </div>
                            </div>

                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Description
                                </label>
                                <label className="mt-2 flex min-h-20 w-full items-center rounded-2xl border border-border bg-background px-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)] transition hover:border-primary/60 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary">
                                    <input
                                        type="text"
                                        name="description"
                                        className="min-w-0 grow border-0 bg-transparent outline-none"
                                        placeholder="Description"
                                        value={formData.description}
                                        onChange={handleFormDataChange}
                                    />
                                </label>
                            </div>

                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Currency
                                </label>
                                <div className="mt-2">
                                    <ExpenseFormPicker
                                        label="Currency"
                                        emptyLabel="Choose currency"
                                        value={formData.currency}
                                        onChange={(currency) =>
                                            setFormData((current) => ({
                                                ...current,
                                                currency,
                                            }))
                                        }
                                        options={currencyOptions}
                                    />
                                </div>
                            </div>
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Amount
                                </label>
                                <label className="mt-2 flex min-h-20 w-full items-center rounded-2xl border border-border bg-background px-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)] transition hover:border-primary/60 focus-within:border-primary focus-within:outline-none focus-within:ring-2 focus-within:ring-primary">
                                    <input
                                        type="number"
                                        name="total"
                                        className="min-w-0 grow border-0 bg-transparent outline-none"
                                        step="0.001"
                                        placeholder="0.00"
                                        value={formData.total}
                                        onChange={handleFormDataChange}
                                        required
                                        min={0}
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

                        <div className="mt-6">
                            <div className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                Split rule
                            </div>
                            <div className="mt-3">
                                {groupMembers.length <= 1 ? (
                                    <></>
                                ) : groupMembers.length === 2 ? (
                                    <ExpenseFormPicker
                                        label="Split rule"
                                        emptyLabel="Choose a split rule"
                                        value={formData.splitRule}
                                        onChange={handleTwoPersonRuleChange}
                                        options={[
                                            {
                                                value: Rule.YouHalf,
                                                label: "You paid, split equally",
                                            },
                                            {
                                                value: Rule.YouFull,
                                                label: "You are owed the full amount",
                                            },
                                            {
                                                value: Rule.OtherHalf,
                                                label: `${groupMembers[0].username} paid, split equally`,
                                            },
                                            {
                                                value: Rule.OtherFull,
                                                label: `${groupMembers[0].username} is owed the full amount`,
                                            },
                                            {
                                                value: Rule.Unequally,
                                                label: "Unequally",
                                            },
                                        ]}
                                    />
                                ) : (
                                    <div className="grid gap-3 md:grid-cols-2">
                                        <ExpenseFormPicker
                                            label="Paid by"
                                            emptyLabel="Choose a payer"
                                            icon={mdiAccountOutline}
                                            value={formData.payerUserId}
                                            onChange={(payerUserId) =>
                                                setFormData((current) => ({
                                                    ...current,
                                                    payerUserId,
                                                }))
                                            }
                                            options={payerOptions}
                                        />
                                        <ExpenseFormPicker
                                            label="Split"
                                            emptyLabel="Choose a split"
                                            value={formData.splitRule}
                                            onChange={(value) =>
                                                setFormData((current) => ({
                                                    ...current,
                                                    splitRule: value as Rule,
                                                }))
                                            }
                                            options={[
                                                {
                                                    value: Rule.Equally,
                                                    label: "Equally",
                                                },
                                                {
                                                    value: Rule.Unequally,
                                                    label: "Unequally",
                                                },
                                            ]}
                                        />
                                    </div>
                                )}
                            </div>
                        </div>

                        <div
                            className={`${
                                formData.splitRule === Rule.Unequally
                                    ? ""
                                    : "hidden"
                            } mt-6 space-y-3`}
                        >
                            {formData.ledgers.map((ledger, index) => (
                                <div
                                    className="flex flex-col gap-2 rounded-2xl border border-border bg-background px-4 py-3 sm:flex-row sm:items-center"
                                    key={ledger.id}
                                >
                                    <p className="sm:w-1/3 text-sm text-foreground/70">
                                        {
                                            groupMembers.find(
                                                (member) =>
                                                    member.userId ===
                                                    ledger.userId
                                            )?.username
                                        }
                                    </p>

                                    <label className="ui-input-shell flex items-center gap-2 w-full sm:w-2/3 bg-background">
                                        Share:
                                        <input
                                            type="number"
                                            className="grow"
                                            step="0.001"
                                            placeholder="0.00"
                                            value={ledger.share}
                                            onChange={(e) => {
                                                const updated = [
                                                    ...formData.ledgers,
                                                ];
                                                updated[index].share =
                                                    parseFloat(
                                                        e.target.value
                                                    ) || 0;
                                                setFormData((prev) => ({
                                                    ...prev,
                                                    ledgers: updated,
                                                }));
                                            }}
                                        />
                                    </label>
                                </div>
                            ))}
                            <div className="text-center">
                                <p
                                    className={`text-sm ${
                                        ledgerShareOk
                                            ? "text-green-700"
                                            : "text-red-700"
                                    }`}
                                >
                                    {ledgerShareMessage}
                                </p>
                            </div>
                        </div>

                        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                            <button
                                type="submit"
                                className="ui-button ui-button-primary w-full sm:w-auto"
                                {...(dataOk ? {} : { disabled: true })}
                            >
                                <Icon path={mdiCheckBold} size={1} />
                                Save changes
                            </button>
                            {indicatorShow && (
                                <span className="ui-spinner ui-spinner-sm"></span>
                            )}
                        </div>
                </form>
            </div>
        </div>
    );
};

const EditExpense = () => {
    return (
        <EditExpenseProvider>
            <EditExpenseContent />
        </EditExpenseProvider>
    );
};

export default EditExpense;
