import { useState } from "react";
import { Link } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiDeleteOutline, mdiOpenInNew, mdiPencilOutline } from "@mdi/js";

import {
    AlertDialog,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";

import DesktopBackLink from "../components/DesktopBackLink";
import { useNavigationReady, useReturnNavigation } from "../hooks/navigation";
import MobilePageHeader from "../components/MobilePageHeader";
import { ExpenseDetailProvider } from "../contexts/ExpenseDetailContext";
import { useExpenseDetail } from "../hooks/ExpenseDetailContextHooks";
import { useCurrencies } from "../hooks/useCurrencies";
import { getExpenseTypePresentation } from "../lib/expenseCategoryPresentation";
import { currencyAmountDigits, formatMoney } from "../lib/money";
import type { ExpenseDetailData } from "../types/expense";
import type { ItemData } from "../types/item";

// Enable only when the complete OCR receipt experience is ready.
const SHOW_RECEIPT_BREAKDOWN = false;

function displayMemberName(
    userId: string,
    username: string,
    currentUserId: string
) {
    return userId === currentUserId ? "You" : username;
}

function formatExpenseMoney(
    value: string | null | undefined,
    currency: string,
    amountDigits: number | null
) {
    if (!value) return null;
    return amountDigits === null
        ? `${value} ${currency}`
        : formatMoney(value, amountDigits, currency);
}

function orderedItems(items: ItemData[]) {
    return items
        .map((item, originalIndex) => ({ item, originalIndex }))
        .sort(
            (left, right) =>
                (left.item.position ?? left.originalIndex) -
                (right.item.position ?? right.originalIndex)
        )
        .map(({ item }) => item);
}

function itemSupportingDetails(
    item: ItemData,
    currency: string,
    amountDigits: number | null
) {
    const quantityAndUnit = [item.quantity, item.unit]
        .filter((value): value is string => Boolean(value))
        .join(" ");
    const unitPrice = formatExpenseMoney(
        item.unitPrice,
        currency,
        amountDigits
    );

    return [
        quantityAndUnit || null,
        unitPrice ? `${unitPrice} each` : null,
    ].filter((value): value is string => Boolean(value));
}

const ExpenseDetailContent = () => {
    const {
        expenseDetail,
        formattedDate,
        expenseId,
        loading,
        errorMessage,
    } = useExpenseDetail();
    const { currencies } = useCurrencies();
    useNavigationReady(!loading);
    const back = useReturnNavigation(expenseDetail?.groupId ? `/group/${expenseDetail.groupId}` : "/");

    if (!expenseId) {
        return <ExpensePageMessage title="Expense ID not found" />;
    }

    if (loading) {
        return <ExpensePageMessage title="Loading expense" loading />;
    }

    if (errorMessage || !expenseDetail) {
        return (
            <ExpensePageMessage
                title="Unable to load expense"
                message={errorMessage ?? "This expense is unavailable."}
            />
        );
    }

    const categoryPresentation = getExpenseTypePresentation(
        expenseDetail.expenseCategory,
        expenseDetail.expenseType
    );
    const amountDigits = currencyAmountDigits(
        currencies,
        expenseDetail.currency
    );
    const title = expenseDetail.description || expenseDetail.expenseType;
    const editRoute = `/expense/${expenseId}/edit`;

    return (
        <main className="page-shell compact-mobile-page expense-detail-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title={title}
                    backTo={`/group/${expenseDetail.groupId}`}
                    backLabel="Back to group"
                    titleIcon={
                        <span
                            aria-hidden="true"
                            className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${categoryPresentation.iconClassName}`}
                        >
                            <Icon path={categoryPresentation.icon} size={0.8} />
                        </span>
                    }
                    action={
                        <Button asChild size="icon" className="size-12">
                            <Link to={editRoute} aria-label="Edit expense">
                                <Icon
                                    path={mdiPencilOutline}
                                    size={0.9}
                                    aria-hidden="true"
                                />
                            </Link>
                        </Button>
                    }
                />

                <DesktopBackLink
                    to={`/group/${expenseDetail.groupId}`}
                    label="Back to group"
                />
                {back.to !== `/group/${expenseDetail.groupId}` ? (
                    <Link className="ui-button ui-button-ghost self-start" to={`/group/${expenseDetail.groupId}`}>View group</Link>
                ) : null}

                <header className="page-header desktop-page-header">
                    <div className="page-header__copy min-w-0">
                        <div className="page-eyebrow">Expense</div>
                        <div className="mt-2 flex min-w-0 items-center gap-4">
                            <span
                                aria-hidden="true"
                                data-testid="expense-category-icon"
                                className={`flex size-14 shrink-0 items-center justify-center rounded-2xl ${categoryPresentation.iconClassName}`}
                            >
                                <Icon
                                    path={categoryPresentation.icon}
                                    size={1.45}
                                />
                            </span>
                            <h1 className="page-title min-w-0">{title}</h1>
                        </div>
                        <p className="page-copy">
                            Review the payment, split, and saved receipt details.
                        </p>
                    </div>
                    <div className="page-actions w-full sm:w-auto">
                        <Button
                            asChild
                            size="lg"
                            className="min-h-11 w-full sm:w-auto"
                        >
                            <Link to={editRoute}>Edit expense</Link>
                        </Button>
                    </div>
                </header>

                <div className="flex flex-col gap-4 md:gap-6">
                    <ExpenseSummary
                        expenseDetail={expenseDetail}
                        formattedDate={formattedDate}
                        amountDigits={amountDigits}
                    />

                    <div
                        className={`grid gap-4 md:gap-6 ${
                            expenseDetail.invoiceUrl
                                ? "lg:grid-cols-[minmax(0,1.35fr)_minmax(18rem,0.65fr)] lg:items-start"
                                : "lg:max-w-3xl"
                        }`}
                    >
                        <div className="flex min-w-0 flex-col gap-4 md:gap-6">
                            <PaymentDetails
                                expenseDetail={expenseDetail}
                                amountDigits={amountDigits}
                            />
                            {SHOW_RECEIPT_BREAKDOWN ? <ReceiptDetails
                                expenseDetail={expenseDetail}
                                amountDigits={amountDigits}
                            /> : null}
                        </div>

                        {expenseDetail.invoiceUrl ? (
                            <aside className="min-w-0">
                                <ReceiptImage
                                    invoiceUrl={expenseDetail.invoiceUrl}
                                />
                            </aside>
                        ) : null}
                    </div>

                    <DangerZone />
                </div>
            </div>
        </main>
    );
};

function ExpensePageMessage({
    title,
    message,
    loading = false,
}: {
    title: string;
    message?: string;
    loading?: boolean;
}) {
    return (
        <main className="page-shell">
            <div className="page-container max-w-4xl">
                <Card
                    className="panel-card min-h-56 items-center justify-center rounded-[2rem] text-center"
                    role={loading ? "status" : "alert"}
                    aria-busy={loading || undefined}
                >
                    <CardHeader className="items-center">
                        {loading ? (
                            <Spinner
                                className="size-8"
                                role="presentation"
                                aria-hidden="true"
                            />
                        ) : null}
                        <CardTitle>
                            <h1>{title}</h1>
                        </CardTitle>
                        {message ? (
                            <CardDescription>{message}</CardDescription>
                        ) : null}
                    </CardHeader>
                    {!loading ? (
                        <CardContent>
                            <Button asChild variant="outline" size="lg">
                                <Link to="/">Back to groups</Link>
                            </Button>
                        </CardContent>
                    ) : null}
                </Card>
            </div>
        </main>
    );
}

export default function ExpenseDetail() {
    return (
        <ExpenseDetailProvider>
            <ExpenseDetailContent />
        </ExpenseDetailProvider>
    );
}

function ExpenseSummary({
    expenseDetail,
    formattedDate,
    amountDigits,
}: {
    expenseDetail: ExpenseDetailData;
    formattedDate: string;
    amountDigits: number | null;
}) {
    const categoryLabel = [
        expenseDetail.expenseCategory,
        expenseDetail.expenseType,
    ]
        .filter(
            (value, index, values) =>
                Boolean(value) && values.indexOf(value) === index
        )
        .join(" · ");

    return (
        <section aria-labelledby="expense-summary-title">
            <Card className="panel-card gap-5 rounded-[2rem] py-5 sm:py-6 md:py-8">
                <CardHeader className="gap-2 px-5 sm:px-6 md:px-8">
                    <div className="section-label">Expense summary</div>
                    <CardTitle>
                        <h2 id="expense-summary-title">Total</h2>
                    </CardTitle>
                    <CardDescription>
                        The recorded amount for this expense.
                    </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-5 px-5 sm:px-6 md:px-8">
                    <div className="text-4xl font-bold tracking-[-0.04em] text-primary tabular-nums sm:text-5xl">
                        {formatExpenseMoney(
                            expenseDetail.total,
                            expenseDetail.currency,
                            amountDigits
                        )}
                    </div>

                    <dl className="grid gap-3 sm:grid-cols-3">
                        <SummaryDetail
                            label="Expense date"
                            value={formattedDate}
                        />
                        <SummaryDetail
                            label="Added by"
                            value={expenseDetail.createdByUsername}
                        />
                        <div className="metric-card flex min-w-0 flex-col gap-2 rounded-2xl p-4">
                            <dt className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                                Category
                            </dt>
                            <dd>
                                <Badge
                                    variant="secondary"
                                    className="h-auto max-w-full whitespace-normal py-1 text-left leading-4"
                                >
                                    {categoryLabel}
                                </Badge>
                            </dd>
                        </div>
                    </dl>
                </CardContent>
            </Card>
        </section>
    );
}

function SummaryDetail({ label, value }: { label: string; value: string }) {
    return (
        <div className="metric-card flex min-w-0 flex-col gap-2 rounded-2xl p-4">
            <dt className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                {label}
            </dt>
            <dd className="truncate font-semibold text-foreground">{value}</dd>
        </div>
    );
}

function PaymentDetails({
    expenseDetail,
    amountDigits,
}: {
    expenseDetail: ExpenseDetailData;
    amountDigits: number | null;
}) {
    const payers = Array.from(
        new Map(
            expenseDetail.ledgers.map((ledger) => [
                ledger.lenderUserId,
                displayMemberName(
                    ledger.lenderUserId,
                    ledger.lenderUsername,
                    expenseDetail.currentUser
                ),
            ])
        ).values()
    );

    return (
        <section aria-labelledby="payment-details-title">
            <Card className="panel-card rounded-[2rem]">
                <CardHeader>
                    <div className="section-label">Payment &amp; split</div>
                    <CardTitle>
                        <h2 id="payment-details-title">
                            {payers.length > 0
                                ? `Paid by ${payers.join(" and ")}`
                                : "Split details"}
                        </h2>
                    </CardTitle>
                    <CardDescription>
                        {payers.length > 0
                            ? "The recorded amount each participant owes."
                            : "No payment relationships were saved for this expense."}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {expenseDetail.ledgers.length > 0 ? (
                        <ul className="flex flex-col gap-2">
                            {expenseDetail.ledgers.map((ledger) => {
                                const borrower = displayMemberName(
                                    ledger.borrowerUserId,
                                    ledger.borrowerUsername,
                                    expenseDetail.currentUser
                                );
                                const lender = displayMemberName(
                                    ledger.lenderUserId,
                                    ledger.lenderUsername,
                                    expenseDetail.currentUser
                                );
                                const relationship =
                                    borrower === "You"
                                        ? `You owe ${lender}`
                                        : lender === "You"
                                          ? `${borrower} owes you`
                                          : `${borrower} owes ${lender}`;

                                return (
                                    <li
                                        className="metric-card flex min-h-14 items-center justify-between gap-3 rounded-2xl px-4 py-3"
                                        key={ledger.id}
                                    >
                                        <span className="min-w-0 font-medium">
                                            {relationship}
                                        </span>
                                        <span className="shrink-0 font-semibold tabular-nums">
                                            {formatExpenseMoney(
                                                ledger.share,
                                                expenseDetail.currency,
                                                amountDigits
                                            )}
                                        </span>
                                    </li>
                                );
                            })}
                        </ul>
                    ) : (
                        <div className="metric-card rounded-2xl p-4 text-sm text-muted-foreground">
                            No split details are available.
                        </div>
                    )}
                </CardContent>
            </Card>
        </section>
    );
}

function ReceiptDetails({
    expenseDetail,
    amountDigits,
}: {
    expenseDetail: ExpenseDetailData;
    amountDigits: number | null;
}) {
    const items = orderedItems(expenseDetail.items);
    const subtotal = formatExpenseMoney(
        expenseDetail.subTotal,
        expenseDetail.currency,
        amountDigits
    );
    const taxFeeTip = formatExpenseMoney(
        expenseDetail.taxFeeTip,
        expenseDetail.currency,
        amountDigits
    );
    const hasTaxFeeTip = Number(expenseDetail.taxFeeTip) !== 0;

    return (
        <section aria-labelledby="receipt-details-title">
            <Card className="panel-card rounded-[2rem]">
                <CardHeader>
                    <div className="section-label">Receipt details</div>
                    <CardTitle>
                        <h2 id="receipt-details-title">Itemized breakdown</h2>
                    </CardTitle>
                    <CardDescription>
                        Saved receipt totals and item information.
                    </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-5">
                    <dl className="flex flex-col gap-3">
                        {subtotal ? (
                            <MoneyRow label="Subtotal" value={subtotal} />
                        ) : null}
                        {hasTaxFeeTip && taxFeeTip ? (
                            <MoneyRow label="Tax & tip" value={taxFeeTip} />
                        ) : null}
                        <Separator />
                        <MoneyRow
                            label="Total"
                            value={
                                formatExpenseMoney(
                                    expenseDetail.total,
                                    expenseDetail.currency,
                                    amountDigits
                                ) ?? expenseDetail.total
                            }
                            emphasized
                        />
                    </dl>

                    <Separator />

                    {items.length > 0 ? (
                        <ol className="flex flex-col gap-2">
                            {items.map((item) => {
                                const supportingDetails =
                                    itemSupportingDetails(
                                        item,
                                        expenseDetail.currency,
                                        amountDigits
                                    );
                                const itemTotal = formatExpenseMoney(
                                    item.lineTotal || item.itemSubTotal,
                                    expenseDetail.currency,
                                    amountDigits
                                );

                                return (
                                    <li
                                        className="metric-card flex min-h-16 items-start justify-between gap-3 rounded-2xl p-4"
                                        key={item.itemId}
                                    >
                                        <span className="min-w-0">
                                            <span className="block font-semibold">
                                                {item.description ||
                                                    item.itemName}
                                            </span>
                                            {supportingDetails.length > 0 ? (
                                                <span className="mt-1 block text-sm text-muted-foreground">
                                                    {supportingDetails.join(
                                                        " · "
                                                    )}
                                                </span>
                                            ) : null}
                                        </span>
                                        {itemTotal ? (
                                            <span className="shrink-0 font-semibold tabular-nums">
                                                {itemTotal}
                                            </span>
                                        ) : null}
                                    </li>
                                );
                            })}
                        </ol>
                    ) : (
                        <div className="metric-card rounded-2xl p-4 text-sm text-muted-foreground">
                            No itemized receipt details were saved.
                        </div>
                    )}
                </CardContent>
            </Card>
        </section>
    );
}

function MoneyRow({
    label,
    value,
    emphasized = false,
}: {
    label: string;
    value: string;
    emphasized?: boolean;
}) {
    return (
        <div className="flex items-baseline justify-between gap-3">
            <dt
                className={
                    emphasized
                        ? "font-semibold text-foreground"
                        : "text-sm text-muted-foreground"
                }
            >
                {label}
            </dt>
            <dd
                className={
                    emphasized
                        ? "font-bold text-foreground tabular-nums"
                        : "font-semibold text-foreground tabular-nums"
                }
            >
                {value}
            </dd>
        </div>
    );
}

function ReceiptImage({ invoiceUrl }: { invoiceUrl: string }) {
    return (
        <section aria-labelledby="receipt-image-title">
            <Card className="panel-card rounded-[2rem]">
                <CardHeader>
                    <div className="section-label">Receipt</div>
                    <CardTitle>
                        <h2 id="receipt-image-title">Receipt image</h2>
                    </CardTitle>
                    <CardDescription>
                        Open the saved image in a new browser tab.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Button
                        asChild
                        variant="outline"
                        size="lg"
                        className="min-h-11 w-full"
                    >
                        <a
                            href={invoiceUrl}
                            target="_blank"
                            rel="noreferrer"
                        >
                            <Icon
                                path={mdiOpenInNew}
                                size={0.8}
                                data-icon="inline-start"
                                aria-hidden="true"
                            />
                            View receipt image
                        </a>
                    </Button>
                </CardContent>
            </Card>
        </section>
    );
}

function DangerZone() {
    const { handleDeleteExpense } = useExpenseDetail();
    const [deleteOpen, setDeleteOpen] = useState(false);
    const [deleting, setDeleting] = useState(false);

    const deleteExpense = async () => {
        setDeleting(true);
        try {
            await handleDeleteExpense();
        } finally {
            setDeleting(false);
        }
    };

    return (
        <section
            aria-labelledby="danger-zone-title"
            className="flex flex-col gap-3 border-t border-border pt-4 sm:flex-row sm:items-center sm:justify-between"
        >
            <div>
                <h2 id="danger-zone-title" className="text-sm font-medium">
                    Remove expense
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">
                    Removing this expense updates the group&apos;s active balances.
                </p>
            </div>
            <Button
                variant="outline"
                size="lg"
                className="min-h-11 w-full text-destructive hover:text-destructive sm:w-auto"
                onClick={() => setDeleteOpen(true)}
            >
                <Icon
                    path={mdiDeleteOutline}
                    size={0.85}
                    data-icon="inline-start"
                    aria-hidden="true"
                />
                Delete expense
            </Button>

            <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
                <AlertDialogContent aria-busy={deleting}>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete this expense?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This removes the expense from the group&apos;s active
                            expense list and recalculates its balances. The
                            retained record is not permanently purged.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={deleting}>
                            Cancel
                        </AlertDialogCancel>
                        <Button
                            variant="destructive"
                            disabled={deleting}
                            onClick={() => void deleteExpense()}
                        >
                            {deleting ? (
                                <Spinner
                                    role="presentation"
                                    aria-hidden="true"
                                    data-icon="inline-start"
                                />
                            ) : null}
                            {deleting
                                ? "Deleting expense"
                                : "Delete expense"}
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </section>
    );
}
