import React, { ReactElement, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiCamera, mdiCheckBold, mdiCheckCircleOutline } from "@mdi/js";

import { API_URL } from "../configs/config";
import {
    ExpenseDetailData,
    ExpenseTypeItem,
    ExpenseUpdateData,
} from "../types/expense";
import { GroupListItem, GroupMember } from "../types/group";
import { LedgerUpdateData } from "../types/ledger";
import { Rule } from "../types/splitRule";

interface expenseFormData {
    groupId: string;
    expenseType: string;
    description: string;
    currency: string;
    total: number;
    splitRule: Rule;

    payerUserId: string;
    ledgers: {
        id: string;
        userId: string;
        share: number;
    }[];
}

const emptyData: expenseFormData = {
    groupId: "",
    expenseType: "",
    description: "",
    total: 0,
    currency: "",
    splitRule: Rule.Equally,
    payerUserId: "",
    ledgers: [],
};

const EditExpense = () => {
    const { id: expenseId = "" } = useParams();

    // handle form submission
    const [feedback, setFeedback] = useState<string>("");
    const [indicatorShow, setIndicatorShow] = useState<boolean>(false);

    const [formData, setFormData] = useState<expenseFormData>(emptyData);

    const handleUpdateExpense = async (e: React.FormEvent) => {
        e.preventDefault();

        try {
            setIndicatorShow(true);

            setIndicatorShow(true);
            setFeedback("");

            // set up ledgers in defult split rules
            const currencyPrecision: Record<"CAD" | "USD" | "NTD", number> = {
                CAD: 2,
                USD: 2,
                NTD: 0,
            };
            const precision: number =
                currencyPrecision[
                    formData.currency as keyof typeof currencyPrecision
                ];

            switch (formData.splitRule) {
                case Rule.Equally:
                case Rule.YouHalf:
                case Rule.OtherHalf: {
                    const peopleCount: number = formData.ledgers.length;

                    const split: number =
                        Math.floor(
                            (formData.total / peopleCount) * 10 ** precision
                        ) /
                        10 ** precision;
                    const remaining: number =
                        formData.total - split * (peopleCount - 1);

                    const randIndex = Math.floor(Math.random() * peopleCount);
                    for (let i = 0; i < peopleCount; i++) {
                        formData.ledgers[i].share =
                            i === randIndex ? remaining : split;
                    }
                    break;
                }

                case Rule.YouFull:
                    formData.ledgers[0].share = 0;
                    formData.ledgers[1].share = formData.total;
                    break;

                case Rule.OtherFull:
                    formData.ledgers[0].share = formData.total;
                    formData.ledgers[1].share = 0;
                    break;

                default:
                    break;
            }

            const payload: ExpenseUpdateData = {
                description: formData.description,
                groupId: formData.groupId,
                payByUserId: formData.payerUserId,
                expTypeId: formData.expenseType,
                total: formData.total.toFixed(precision),
                currency: formData.currency,
                splitRule: formData.splitRule,
                ledgers: formData.ledgers.map(
                    (ledger) =>
                        ({
                            ledgerId: ledger.id,
                            borrowerUserId: ledger.userId,
                            lenderUserId: formData.payerUserId,
                            share: ledger.share.toFixed(precision),
                        } as LedgerUpdateData)
                ),
            };

            const response = await fetch(`${API_URL}/expense/${expenseId}`, {
                method: "PUT",
                credentials: "include",
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                const errorMessage = await response.text();
                throw new Error(`Failed to update expense: ${errorMessage}`);
            }

            window.location.href = `/expense/${expenseId}`;
        } catch (error) {
            console.error("Error updating expense:", error);
            setFeedback("Error updating expense");
        } finally {
            setIndicatorShow(false);
        }
    };

    // handle page load
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expTypeOptions, setExpTypeOptions] = useState<ReactElement[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);

    useEffect(() => {
        const fetchGroupList = async () => {
            const response = await fetch(`${API_URL}/groups`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: GroupListItem[] = await response.json();

            setGroupList(data);
            setFormData((prev) => ({
                ...prev,
                groupId: data[0].id,
            }));
        };

        const fetchExpeseTypes = async () => {
            const response = await fetch(`${API_URL}/expense_types`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: ExpenseTypeItem[] = await response.json();
            // create options
            const options: ReactElement[] = [];
            let lastCategory = "";
            data.forEach((type) => {
                if (lastCategory !== type.category) {
                    options.push(
                        <option
                            disabled
                            key={type.category}
                            className="text-lg font-black font-mono"
                        >
                            {type.category}
                        </option>
                    );
                    lastCategory = type.category;
                }

                options.push(
                    <option value={type.id} key={type.id}>
                        {type.name}
                    </option>
                );
            });
            setExpTypeOptions(options);
        };

        const fetchExpenseDetail = async () => {
            const response = await fetch(`${API_URL}/expense/${expenseId}`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: ExpenseDetailData = await response.json();

            setFormData((prev) => ({
                ...prev,
                groupId: data.groupId,
                expenseType: data.expenseTypeId,
                description: data.description,
                total: parseFloat(data.total),
                currency: data.currency,
                splitRule: data.splitRule as Rule,
                payerUserId: data.ledgers[0].lenderUserId,
                ledgers: data.ledgers.map((ledger) => ({
                    id: ledger.id,
                    userId: ledger.borrowerUserId,
                    share: parseFloat(ledger.share),
                })),
            }));

            const responseGroupMember = await fetch(
                `${API_URL}/group_member/${data.groupId}`,
                {
                    method: "GET",
                    credentials: "include",
                }
            );
            if (!responseGroupMember.ok) return;

            const groupMember: GroupMember[] = await responseGroupMember.json();

            setGroupMembers(groupMember);
        };

        fetchGroupList();
        fetchExpeseTypes();
        fetchExpenseDetail();
    }, []);

    // handle form data update
    const handleFormDataChange = (
        e: React.ChangeEvent<HTMLSelectElement | HTMLInputElement>
    ) => {
        const { name, value } = e.target;
        setFormData((prev) => ({
            ...prev,
            [name]: value,
        }));
    };

    // handle form data validation
    const [ledgerShareOk, setLedgerShareOk] = useState<boolean>(false);
    const [ledgerShareMessage, setLedgerShareMessage] = useState<string>("");
    const [dataOk, setDataOk] = useState<boolean>(false);

    useEffect(() => {
        const totalOk = formData.total > 0;
        const descriptionOk = formData.description.length > 0;

        if (formData.splitRule !== Rule.Unequally) {
            setDataOk(totalOk && descriptionOk);
            return;
        }

        const ledgerTotal = formData.ledgers.reduce(
            (acc, ledger) => acc + ledger.share,
            0
        );
        const ledgerOk =
            ledgerTotal === formData.total &&
            formData.ledgers.every((ledger) => ledger.share >= 0);

        setDataOk(totalOk && descriptionOk && ledgerOk);
        setLedgerShareOk(ledgerOk);
        setLedgerShareMessage(
            ledgerOk
                ? `Total $0 ${formData.currency} left.`
                : `Total $${(formData.total - ledgerTotal).toFixed(2)} ${
                      formData.currency
                  } left.`
        );
    }, [formData]);

    return (
        <div className="flex flex-row justify-center items-center py-5 w-screen">
            <form
                className="flex flex-col justify-center items-center py-5 space-y-5 md:w-1/3 w-5/6 max-w-md"
                onSubmit={handleUpdateExpense}
            >
                <div className="text-2xl">Edit Expense</div>

                {/* GROUP SELECTOR */}
                <select
                    className="select select-bordered w-full text-base text-center"
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

                {/* EXPENSE TYPE SELECTOR */}
                <select
                    className="select select-bordered w-full text-base text-center"
                    name="expenseType"
                    value={formData.expenseType}
                    onChange={handleFormDataChange}
                >
                    {expTypeOptions}
                </select>

                {/* DESCRIPTION INPUT */}
                <div className="flex flex-row justify-start items-start w-full">
                    <label className="input input-bordered flex items-center w-full">
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

                <div className="flex flex-row justify-start items-start w-full">
                    {/* CURRENCY SELECTOR */}
                    <select
                        className="select select-bordered w-1/3 text-base text-center"
                        name="currency"
                        value={formData.currency}
                        onChange={handleFormDataChange}
                    >
                        <option>CAD</option>
                        <option>NTD</option>
                        <option>USD</option>
                    </select>
                    <label className="input input-bordered flex items-center w-full">
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

                {/* RECEIPT UPLOAD BUTTON */}
                <div className="hidden">
                    <label
                        style={{ display: "inline-block" }}
                        className="w-2/3 h-12 border border-gray-400 rounded-full bg-base-100 hover:bg-base-300"
                    >
                        <input type="file" style={{ display: "none" }} />
                        <div className="flex flex-row items-center justify-center h-full space-x-3">
                            <Icon path={mdiCamera} size={1} />
                            <p>Upload Receipt</p>
                        </div>
                    </label>
                </div>

                {/* SPLIT RULE SELECTOR */}
                {groupMembers.length <= 1 ? (
                    <></>
                ) : groupMembers.length === 2 ? (
                    // 2 members
                    <>
                        <select
                            className="select select-bordered w-full text-base text-center"
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
                                                groupMembers.length - 1
                                            ].userId;
                                        break;
                                    case Rule.OtherHalf:
                                    case Rule.OtherFull:
                                        payerUserId = groupMembers[0].userId;
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
                                {groupMembers[0].username} paid, split euqally
                            </option>
                            <option value={Rule.OtherFull}>
                                {groupMembers[0].username} is owed the full
                                amount
                            </option>
                            <option value={Rule.Unequally}>Unequally</option>
                        </select>
                        <div className="hidden flex-row justify-center items-center w-full space-x-2">
                            <p className="w-max">Paid by</p>
                            <select
                                className="select select-sm select-bordered w-max border-dashed"
                                name="payer"
                            >
                                <option
                                    value={
                                        groupMembers[groupMembers.length - 1]
                                            ?.userId
                                    }
                                >
                                    You
                                </option>
                                <option value={groupMembers[0].userId}>
                                    {groupMembers[0].userId}
                                </option>
                            </select>
                        </div>
                    </>
                ) : (
                    // N members
                    <div className="flex flex-row justify-center items-center w-full space-x-2">
                        <p className="w-max">Paid by</p>
                        <select
                            className="select select-sm select-bordered w-max border-dashed"
                            name="payerUserId"
                            value={formData.payerUserId}
                            onChange={handleFormDataChange}
                        >
                            <option
                                value={
                                    groupMembers[groupMembers.length - 1].userId
                                }
                                key={
                                    groupMembers[groupMembers.length - 1].userId
                                }
                            >
                                You
                            </option>
                            {groupMembers.map((member) => {
                                if (
                                    member.userId !==
                                    groupMembers[groupMembers.length - 1].userId
                                ) {
                                    return (
                                        <option
                                            value={member.userId}
                                            key={member.userId}
                                        >
                                            {member.username}
                                        </option>
                                    );
                                }
                            })}
                        </select>
                        <p className="w-max">and split</p>
                        <select
                            className="select select-sm select-bordered w-max border-dashed"
                            name="splitRule"
                            value={formData.splitRule}
                            onChange={handleFormDataChange}
                        >
                            <option value={Rule.Equally}>Equally</option>
                            <option value={Rule.Unequally}>Unequally</option>
                        </select>
                    </div>
                )}

                {/* LEDGERS - FOR UNEQUAL SPLIT RULE */}
                <div
                    className={`${
                        formData.splitRule === Rule.Unequally ? "" : "hidden"
                    } flex-col justify-center items-center w-full space-y-1`}
                >
                    {formData.ledgers.map((ledger, index) => (
                        <div
                            className="flex items-center w-full"
                            key={ledger.id}
                        >
                            <p className="w-1/3 text-right mr-2">
                                {
                                    groupMembers.find(
                                        (member) =>
                                            member.userId === ledger.userId
                                    )?.username
                                }
                                :
                            </p>

                            <label className="input input-bordered flex items-center w-2/3 gap-2">
                                Share:
                                <input
                                    type="number"
                                    className="grow"
                                    step="0.001"
                                    placeholder="0.00"
                                    value={ledger.share}
                                    onChange={(e) => {
                                        const updated = [...formData.ledgers];
                                        updated[index].share =
                                            parseFloat(e.target.value) || 0;
                                        setFormData((prev) => ({
                                            ...prev,
                                            ledgers: updated,
                                        }));
                                    }}
                                />
                            </label>
                        </div>
                    ))}
                    <div className="flex flex-col items-center w-full">
                        <p
                            className={`${
                                ledgerShareOk
                                    ? "text-green-700"
                                    : "text-red-700"
                            }`}
                        >
                            {ledgerShareMessage}
                        </p>
                    </div>
                </div>

                {/* SUBMIT BUTTON */}
                <button
                    type="submit"
                    className="btn btn-active btn-neutral btn-wide text-lg font-light"
                    {...(dataOk ? {} : { disabled: true })}
                >
                    <Icon path={mdiCheckBold} size={1} />
                    Update
                </button>

                {/* FEEDBACK */}
                <div className={`${indicatorShow ? "" : "hidden"}`}>
                    <div className="flex justify-center items-center w-full">
                        <span className="loading loading-spinner loading-md"></span>
                    </div>
                </div>
                <div className={`${feedback.length === 0 ? "hidden" : ""}`}>
                    <div className="animate-fade">
                        <div role="alert" className="alert alert-success">
                            <Icon path={mdiCheckCircleOutline} size={1} />
                            <span>{feedback}</span>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    );
};

export default EditExpense;
