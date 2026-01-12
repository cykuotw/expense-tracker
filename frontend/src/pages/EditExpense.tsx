import Icon from "@mdi/react";
import { Link } from "react-router-dom";
import { mdiCamera, mdiCheckBold, mdiSubdirectoryArrowLeft } from "@mdi/js";

import { Rule } from "../types/splitRule";
import { EditExpenseProvider } from "../contexts/EditExpenseContext";
import { useEditExpense } from "../hooks/EditExpenseContextHooks";

const EditExpenseContent = () => {
    const {
        formData,
        setFormData,
        groupList,
        expTypeOptions,
        groupMembers,
        indicatorShow,
        dataOk,
        ledgerShareOk,
        ledgerShareMessage,
        handleUpdateExpense,
        handleFormDataChange,
    } = useEditExpense();

    const expenseId = window.location.pathname.split("/")[2] || "";

    return (
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-28 md:pb-0">
            <div className="mx-auto w-full max-w-5xl px-4 py-10 md:py-14">
                <div className="flex flex-col gap-8">
                    <div className="space-y-3">
                        <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                            Expense
                        </div>
                        <h1 className="text-3xl font-semibold md:text-4xl">
                            Edit expense
                        </h1>
                        <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                            Update details and adjust how this expense is split.
                        </p>
                    </div>

                    <form
                        className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm"
                        onSubmit={handleUpdateExpense}
                    >
                        <div className="grid gap-5 md:grid-cols-2">
                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Group
                                </label>
                                <select
                                    className="select select-bordered mt-2 w-full text-base"
                                    name="groupId"
                                    value={formData.groupId}
                                    onChange={handleFormDataChange}
                                >
                                    {groupList.map((group) => (
                                        <option key={group.id} value={group.id}>
                                            {group.groupName}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Expense type
                                </label>
                                <select
                                    className="select select-bordered mt-2 w-full text-base"
                                    name="expenseType"
                                    value={formData.expenseType}
                                    onChange={handleFormDataChange}
                                >
                                    {expTypeOptions}
                                </select>
                            </div>

                            <div className="md:col-span-2">
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Description
                                </label>
                                <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                    <input
                                        type="text"
                                        name="description"
                                        className="grow"
                                        placeholder="Description"
                                        value={formData.description}
                                        onChange={handleFormDataChange}
                                    />
                                </label>
                            </div>

                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Currency
                                </label>
                                <select
                                    className="select select-bordered mt-2 w-full text-base"
                                    name="currency"
                                    value={formData.currency}
                                    onChange={handleFormDataChange}
                                >
                                    <option>CAD</option>
                                    <option>NTD</option>
                                    <option>USD</option>
                                </select>
                            </div>
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Amount
                                </label>
                                <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                    <input
                                        type="number"
                                        name="total"
                                        className="grow"
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
                                className="w-2/3 h-12 border border-gray-400 rounded-full bg-base-100 hover:bg-base-300"
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
                            <div className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                Split rule
                            </div>
                            <div className="mt-3">
                                {groupMembers.length <= 1 ? (
                                    <></>
                                ) : groupMembers.length === 2 ? (
                                    <select
                                        className="select select-bordered w-full text-base"
                                        name="splitRule"
                                        value={formData.splitRule}
                                        onChange={(e) => {
                                            handleFormDataChange(e);

                                            let payerUserId = "";
                                            switch (e.target.value) {
                                                case Rule.YouHalf:
                                                case Rule.YouFull:
                                                    payerUserId =
                                                        groupMembers[
                                                            groupMembers.length -
                                                                1
                                                        ].userId;
                                                    break;
                                                case Rule.OtherHalf:
                                                case Rule.OtherFull:
                                                    payerUserId =
                                                        groupMembers[0].userId;
                                                    break;
                                                default:
                                                    break;
                                            }
                                            setFormData((prev) => ({
                                                ...prev,
                                                payerUserId: payerUserId,
                                            }));
                                        }}
                                    >
                                        <option value={Rule.YouHalf}>
                                            You paid, split equally
                                        </option>
                                        <option value={Rule.YouFull}>
                                            You are owed the full amount
                                        </option>
                                        <option value={Rule.OtherHalf}>
                                            {groupMembers[0].username} paid,
                                            split equally
                                        </option>
                                        <option value={Rule.OtherFull}>
                                            {groupMembers[0].username} is owed
                                            the full amount
                                        </option>
                                        <option value={Rule.Unequally}>
                                            Unequally
                                        </option>
                                    </select>
                                ) : (
                                    <div className="flex flex-col gap-3 md:flex-row md:items-center">
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-base-content/70">
                                                Paid by
                                            </span>
                                            <select
                                                className="select select-sm select-bordered border-dashed"
                                                name="payerUserId"
                                                value={formData.payerUserId}
                                                onChange={handleFormDataChange}
                                            >
                                                <option
                                                    value={
                                                        groupMembers[
                                                            groupMembers.length -
                                                                1
                                                        ].userId
                                                    }
                                                    key={
                                                        groupMembers[
                                                            groupMembers.length -
                                                                1
                                                        ].userId
                                                    }
                                                >
                                                    You
                                                </option>
                                                {groupMembers.map((member) => {
                                                    if (
                                                        member.userId !==
                                                        groupMembers[
                                                            groupMembers.length -
                                                                1
                                                        ].userId
                                                    ) {
                                                        return (
                                                            <option
                                                                value={
                                                                    member.userId
                                                                }
                                                                key={
                                                                    member.userId
                                                                }
                                                            >
                                                                {
                                                                    member.username
                                                                }
                                                            </option>
                                                        );
                                                    }
                                                })}
                                            </select>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-base-content/70">
                                                Split
                                            </span>
                                            <select
                                                className="select select-sm select-bordered border-dashed"
                                                name="splitRule"
                                                value={formData.splitRule}
                                                onChange={handleFormDataChange}
                                            >
                                                <option value={Rule.Equally}>
                                                    Equally
                                                </option>
                                                <option value={Rule.Unequally}>
                                                    Unequally
                                                </option>
                                            </select>
                                        </div>
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
                                    className="flex flex-col gap-2 rounded-2xl border border-base-200 bg-base-100 px-4 py-3 sm:flex-row sm:items-center"
                                    key={ledger.id}
                                >
                                    <p className="sm:w-1/3 text-sm text-base-content/70">
                                        {
                                            groupMembers.find(
                                                (member) =>
                                                    member.userId ===
                                                    ledger.userId
                                            )?.username
                                        }
                                    </p>

                                    <label className="input input-bordered flex items-center gap-2 w-full sm:w-2/3 bg-base-100">
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
                                className="btn btn-neutral w-full sm:w-auto"
                                {...(dataOk ? {} : { disabled: true })}
                            >
                                <Icon path={mdiCheckBold} size={1} />
                                Save changes
                            </button>
                            <Link
                                className="btn btn-ghost w-full sm:w-auto"
                                to={`/expense/${expenseId}`}
                            >
                                <Icon
                                    path={mdiSubdirectoryArrowLeft}
                                    size={1}
                                />
                                Cancel
                            </Link>
                            {indicatorShow && (
                                <span className="loading loading-spinner loading-md"></span>
                            )}
                        </div>
                    </form>
                </div>
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
