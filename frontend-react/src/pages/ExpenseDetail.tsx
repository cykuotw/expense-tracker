import React, { useEffect, useMemo, useRef, useState } from "react";
import { useParams, Link } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiSubdirectoryArrowLeft } from "@mdi/js";

import Dropdown from "../components/Dropdown";
import { API_URL } from "../configs/config";

import { ExpenseDetailData } from "../types/expense";

export default function ExpenseDetail() {
    const { id: expenseId = "" } = useParams();

    const [expenseDetail, setExpenseDetail] =
        useState<ExpenseDetailData | null>(null);

    const formattedDate = useMemo(() => {
        if (!expenseDetail?.expenseTime) return "";
        return new Date(expenseDetail.expenseTime).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }, [expenseDetail?.expenseTime]);

    useEffect(() => {
        if (!expenseId) return;

        const fetchExpenseDetail = async () => {
            try {
                const response = await fetch(
                    `${API_URL}/expense/${expenseId}`,
                    {
                        method: "GET",
                        credentials: "include",
                        headers: {
                            "Content-Type": "application/json",
                        },
                    }
                );
                const data: ExpenseDetailData = await response.json();
                setExpenseDetail(data);
            } catch (error) {
                console.log(error);
            }
        };

        fetchExpenseDetail();
    }, [expenseId]);

    if (!expenseId || expenseId.length === 0) {
        return <div>Expense ID not found</div>;
    }

    return (
        <div className="flex flex-col justify-center items-center space-x-1 h-full w-full py-5 px-2">
            <div className="flex justify-center w-full">
                <div className="card bg-base-100 border-2 p-6 w-full max-w-md">
                    <h1 className="text-xl font-bold mb-2">
                        {expenseDetail?.description}
                    </h1>
                    <p className="text-4xl font-semibold text-success mb-4">
                        ${expenseDetail?.total} {expenseDetail?.currency}
                    </p>
                    <p className="text-sm text-gray-600">
                        Added by {expenseDetail?.createdByUsername} on{" "}
                        {formattedDate}
                    </p>
                    <p className="text-sm text-gray-600 mb-6">
                        Category: {expenseDetail?.expenseType}
                    </p>

                    <LedgersDropdown expenseDetail={expenseDetail} />
                    <ItemsDropdown expenseDetail={expenseDetail} />
                    <InvoiceImage expenseDetail={expenseDetail} />

                    <div className="flex justify-between mt-6">
                        <button className="flex items-center space-x-2 text-blue-600 hover:text-blue-800">
                            <Link to={`/expense/${expenseId}/edit`}>
                                <span>Edit Expense</span>
                            </Link>
                        </button>
                        <DeleteBtn
                            groupId={expenseDetail?.groupId || ""}
                            expenseId={expenseId}
                        />
                    </div>
                </div>
            </div>
            <div className="flex justify-center w-full mt-4">
                <Link
                    className="btn btn-ghost"
                    to={`/group/${expenseDetail?.groupId}`}
                >
                    <Icon path={mdiSubdirectoryArrowLeft} size={1} />
                    Back to Group
                </Link>
            </div>
        </div>
    );
}

const LedgersDropdown = ({
    expenseDetail,
}: {
    expenseDetail: ExpenseDetailData | null;
}) => {
    const [isLedgerOpen, setIsLedgerOpen] = useState(false);
    const ledgerDropdown = useRef<HTMLButtonElement>(null);

    return (
        <div className="p-4 rounded-lg">
            <button
                className="flex items-center justify-between w-full font-medium focus:outline-none"
                ref={ledgerDropdown}
                onBlur={(e) => {
                    const isClosed = !ledgerDropdown.current?.contains(
                        e.relatedTarget as Node
                    );
                    setIsLedgerOpen(!isClosed);
                }}
                onClick={(e) => {
                    e.stopPropagation();
                    setIsLedgerOpen(!isLedgerOpen);
                }}
            >
                <span>
                    {expenseDetail?.ledgers[0].lenderUserId ===
                    expenseDetail?.currentUser
                        ? `You paid ${expenseDetail?.total} ${expenseDetail?.currency}`
                        : ` ${expenseDetail?.ledgers[0].lenderUsername} paid ${expenseDetail?.total} ${expenseDetail?.currency}`}
                </span>
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className={`h-5 w-5 transition-transform ${
                        isLedgerOpen ? "rotate-180" : ""
                    }`}
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="2"
                        d="M19 9l-7 7-7-7"
                    ></path>
                </svg>
            </button>
            <ul
                className={`mt-3 ${
                    isLedgerOpen ? "" : "hidden"
                } border-l-2 border-primary pl-1 space-y-2`}
            >
                {expenseDetail?.ledgers.map((ledger) => {
                    return (
                        <li
                            className="relative text-gray-600 pl-2"
                            key={ledger.id}
                        >
                            {ledger.borrowerUsername} owes ${ledger.share}{" "}
                            {expenseDetail?.currency}
                        </li>
                    );
                })}
            </ul>
        </div>
    );
};

const ItemsDropdown = ({
    expenseDetail,
}: {
    expenseDetail: ExpenseDetailData | null;
}) => {
    // TODO: image reconition feature

    return (
        <>
            {Array.isArray(expenseDetail?.items) &&
            expenseDetail?.items.length !== 0 ? (
                <div className="p-4 rounded-lg">
                    <Dropdown
                        label="Items"
                        dropdownType="dropdown-bottom dropdown-start"
                    >
                        {expenseDetail?.items.map((item) => {
                            return <li>{item.itemName}</li>;
                        })}
                    </Dropdown>
                </div>
            ) : null}
        </>
    );
};

const InvoiceImage = ({
    expenseDetail,
}: {
    expenseDetail: ExpenseDetailData | null;
}) => {
    return (
        <>
            {expenseDetail?.invoiceUrl !== "" ? (
                <div className="flex flex-col justify-center items-center py-5 space-y-1">
                    <h1 className="text-xl">Invoice Image</h1>
                    <button className="btn btn-ghost btn-active">
                        View Invoice Image
                    </button>
                    <div id="indicator" className="htmx-indicator">
                        <div className="flex justify-center items-center w-full">
                            <span className="loading loading-spinner loading-md"></span>
                        </div>
                    </div>
                </div>
            ) : null}
        </>
    );
};

const DeleteBtn = ({
    groupId = "",
    expenseId = "",
}: {
    groupId: string;
    expenseId: string;
}) => {
    const handleDeleteExpense = async (e: React.FormEvent) => {
        if (groupId === "" || expenseId === "") return;

        e.preventDefault();

        try {
            const response = await fetch(
                `${API_URL}/delete_expense/${expenseId}`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                }
            );

            if (response.status === 200) {
                window.location.href = `/group/${groupId}`;
            }
        } catch (error) {
            console.log(error);
        }
    };

    return (
        <>
            <button
                className="flex items-center space-x-2 text-red-600 hover:text-red-800"
                onClick={() =>
                    (
                        document.getElementById(
                            "delete_confirm"
                        ) as HTMLDialogElement
                    ).showModal()
                }
            >
                <span>Delete Expense</span>
            </button>
            <dialog id="delete_confirm" className="modal">
                <div className="modal-box">
                    <h3 className="text-lg font-bold">Are You Sure?</h3>
                    <p className="py-4">Expense will be deleted permanently</p>
                    <div className="modal-action">
                        <form method="dialog" className="flex space-x-1">
                            <button
                                className="btn bg-red-600 text-white w-1/2"
                                onClick={(e) => handleDeleteExpense(e)}
                            >
                                Delete
                            </button>
                            <button className="btn w-1/2">Cancel</button>
                        </form>
                    </div>
                </div>
            </dialog>
        </>
    );
};
