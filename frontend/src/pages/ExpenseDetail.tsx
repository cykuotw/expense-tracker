import { useRef, useState } from "react";
import { Link } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiSubdirectoryArrowLeft } from "@mdi/js";

import Dropdown from "../components/Dropdown";

import { ExpenseDetailData } from "../types/expense";
import { ExpenseDetailProvider } from "../contexts/ExpenseDetailContext";
import { useExpenseDetail } from "../hooks/ExpenseDetailContextHooks";

const ExpenseDetailContent = () => {
    const { expenseDetail, formattedDate, expenseId } = useExpenseDetail();

    if (!expenseId || expenseId.length === 0) {
        return <div>Expense ID not found</div>;
    }

    return (
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-28 md:pb-0">
            <div className="mx-auto w-full max-w-4xl px-4 py-10 md:py-14">
                <div className="flex flex-col gap-6">
                    <div className="space-y-3">
                        <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                            Expense Detail
                        </div>
                        <h1 className="text-3xl font-semibold md:text-4xl">
                            {expenseDetail?.description}
                        </h1>
                        <p className="text-sm text-base-content/70 md:text-base">
                            Added by {expenseDetail?.createdByUsername} on{" "}
                            {formattedDate}
                        </p>
                    </div>

                    <div className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                            <div>
                                <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                                    Total
                                </div>
                                <div className="mt-2 text-4xl font-semibold text-success">
                                    ${expenseDetail?.total}{" "}
                                    {expenseDetail?.currency}
                                </div>
                            </div>
                            <div className="rounded-2xl border border-base-200 bg-base-100 px-4 py-3 text-sm text-base-content/70">
                                Category: {expenseDetail?.expenseType}
                            </div>
                        </div>

                        <div className="mt-6 space-y-4">
                            <LedgersDropdown expenseDetail={expenseDetail} />
                            <ItemsDropdown expenseDetail={expenseDetail} />
                            <InvoiceImage expenseDetail={expenseDetail} />
                        </div>

                        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                            <Link
                                to={`/expense/${expenseId}/edit`}
                                className="btn btn-neutral w-full sm:w-auto"
                            >
                                Edit Expense
                            </Link>
                            <DeleteBtn />
                        </div>
                    </div>

                    <div className="flex justify-start">
                        <Link
                            className="btn btn-ghost w-full sm:w-auto"
                            to={`/group/${expenseDetail?.groupId}`}
                        >
                            <Icon path={mdiSubdirectoryArrowLeft} size={1} />
                            Back to Group
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default function ExpenseDetail() {
    return (
        <ExpenseDetailProvider>
            <ExpenseDetailContent />
        </ExpenseDetailProvider>
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
        <div className="rounded-2xl border border-base-200 bg-base-100 p-4">
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
                } border-l-2 border-primary pl-2 space-y-2`}
            >
                {expenseDetail?.ledgers.map((ledger) => {
                    return (
                        <li
                            className="relative text-sm text-base-content/70"
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
                <div className="rounded-2xl border border-base-200 bg-base-100 p-4">
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
                <div className="rounded-2xl border border-base-200 bg-base-100 p-4">
                    <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                        Invoice
                    </div>
                    <button className="btn btn-ghost mt-3">
                        View invoice image
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

const DeleteBtn = () => {
    const { handleDeleteExpense } = useExpenseDetail();

    return (
        <>
            <button
                className="btn btn-ghost w-full sm:w-auto text-error"
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
