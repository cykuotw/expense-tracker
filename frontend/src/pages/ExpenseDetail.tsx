import { useRef, useState } from "react";
import { Link } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiSubdirectoryArrowLeft } from "@mdi/js";

import Dropdown from "../components/Dropdown";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import {
    AlertDialog,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

import { ExpenseDetailData } from "../types/expense";
import { ExpenseDetailProvider } from "../contexts/ExpenseDetailContext";
import { useExpenseDetail } from "../hooks/ExpenseDetailContextHooks";

const ExpenseDetailContent = () => {
    const { expenseDetail, formattedDate, expenseId } = useExpenseDetail();

    if (!expenseId || expenseId.length === 0) {
        return <div>Expense ID not found</div>;
    }

    return (
        <div className="page-shell">
            <div className="page-container max-w-4xl">
                <div className="flex flex-col gap-6">
                    <div className="page-header">
                        <div className="page-header__copy">
                            <div className="page-eyebrow">Expense</div>
                            <h1 className="page-title">
                                {expenseDetail?.description}
                            </h1>
                            <p className="page-copy">
                                Added by {expenseDetail?.createdByUsername} on{" "}
                                {formattedDate}
                            </p>
                        </div>
                    </div>

                    <div className="panel-card rounded-[2rem] p-6 md:p-8">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                            <div>
                                <div className="section-label">
                                    Total
                                </div>
                                <div className="mt-2 text-4xl font-semibold text-success">
                                    ${expenseDetail?.total}{" "}
                                    {expenseDetail?.currency}
                                </div>
                            </div>
                            <div className="metric-card rounded-[1.5rem] px-4 py-3 text-sm text-foreground/70">
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
                                className="ui-button ui-button-primary w-full sm:w-auto"
                            >
                                Edit Expense
                            </Link>
                            <DeleteBtn />
                        </div>
                    </div>

                    <div className="flex justify-start">
                        <Link
                            className="ui-button ui-button-ghost w-full sm:w-auto"
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
        <div className="rounded-2xl border border-border bg-background p-4">
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
                            className="relative text-sm text-foreground/70"
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
                <div className="rounded-2xl border border-border bg-background p-4">
                    <Dropdown
                        label="Items"
                        side="bottom"
                    >
                        {expenseDetail?.items.map((item) => {
                            return (
                                <DropdownMenuItem key={item.itemName}>
                                    {item.itemName}
                                </DropdownMenuItem>
                            );
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
                <div className="rounded-2xl border border-border bg-background p-4">
                    <div className="text-sm font-semibold uppercase tracking-[0.2em] text-foreground/60">
                        Invoice
                    </div>
                    <button className="ui-button ui-button-ghost mt-3">
                        View invoice image
                    </button>
                </div>
            ) : null}
        </>
    );
};

const DeleteBtn = () => {
    const { handleDeleteExpense } = useExpenseDetail();
    const [deleteOpen, setDeleteOpen] = useState(false);

    return (
        <>
            <button
                className="ui-button ui-button-ghost w-full sm:w-auto text-destructive"
                onClick={() => setDeleteOpen(true)}
            >
                <span>Delete Expense</span>
            </button>
            <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete this expense?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This permanently deletes the expense for everyone in
                            the group.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <Button
                            variant="destructive"
                            onClick={() => void handleDeleteExpense()}
                        >
                            Delete expense
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
};
