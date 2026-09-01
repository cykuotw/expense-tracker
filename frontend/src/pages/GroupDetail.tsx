import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import Icon from "@mdi/react";
import {
    mdiAccountMultipleOutline,
    mdiChevronRight,
    mdiHandshakeOutline,
    mdiPencilOutline,
    mdiPlus,
} from "@mdi/js";
import { ExpenseData } from "../types/expense";
import ExpenseCard from "../components/expense/ExpenseCard";
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
import { GroupDetailProvider } from "../contexts/GroupDetailContext";
import { useGroupDetail } from "../hooks/GroupDetailContextHooks";
import { getGroupTypePresentation } from "../lib/groupTypePresentation";
import MobilePageHeader from "../components/MobilePageHeader";

const GroupDetailContent = () => {
    const {
        groupinfo,
        balance,
        unsettledExpenses,
        unsettledLoading,
        unsettledHasMore,
        expenseOrder,
        expenseListRefreshVersion,
        setExpenseOrder,
        settledExpenses,
        settledLoading,
        settledHasMore,
        loading,
        groupId,
        handleSettle,
        loadMoreUnsettledExpenses,
        loadSettledExpenses,
        loadMoreSettledExpenses,
    } = useGroupDetail();
    const [showSettled, setShowSettled] = useState(false);
    const [settleOpen, setSettleOpen] = useState(false);
    const [showAllBalances, setShowAllBalances] = useState(false);
    const groupType = getGroupTypePresentation(groupinfo?.groupType);
    const memberCount = groupinfo?.members?.length ?? 0;
    const loadedSettledExpensesRef = useRef<string | null>(null);
    const balanceEntries = balance
        ? balance.balances.flatMap((entry) => {
              if (entry.receiverUserId === balance.currentUser) {
                  return [
                      {
                          ...entry,
                          label: `${entry.senderUsername} owes you`,
                          tone: "text-success",
                      },
                  ];
              }
              if (entry.senderUserId === balance.currentUser) {
                  return [
                      {
                          ...entry,
                          label: `You owe ${entry.receiverUsername}`,
                          tone: "text-destructive",
                      },
                  ];
              }
              return [];
          })
        : [];
    const visibleMobileBalances = showAllBalances
        ? balanceEntries
        : balanceEntries.slice(0, 2);

    useEffect(() => {
        if (!showSettled) return;
        const requestKey = `${expenseOrder}:${expenseListRefreshVersion}`;
        if (loadedSettledExpensesRef.current === requestKey) return;

        loadedSettledExpensesRef.current = requestKey;
        void loadSettledExpenses();
    }, [
        expenseListRefreshVersion,
        expenseOrder,
        loadSettledExpenses,
        showSettled,
    ]);

    if (loading) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="ui-spinner ui-spinner-lg"></span>
            </div>
        );
    }

    return (
        <div className="page-shell compact-mobile-page group-detail-page">
            <div className="page-container">
                <MobilePageHeader
                    title={groupinfo?.groupName || "Group"}
                    backTo="/"
                    backLabel="Back to groups"
                    titleIcon={
                        <span
                            className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${groupType.iconClassName}`}
                            aria-hidden="true"
                        >
                            <Icon path={groupType.icon} size={0.8} />
                        </span>
                    }
                    action={
                        <Link
                            to={`/group/${groupId}/edit`}
                            className="ui-button ui-button-outline min-h-12 min-w-12 px-3"
                            aria-label="Edit group"
                        >
                            <Icon
                                path={mdiPencilOutline}
                                size={1}
                                aria-hidden="true"
                            />
                        </Link>
                    }
                />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group</div>
                        <div className="mt-2 flex items-center gap-3"><span className={`flex size-11 items-center justify-center rounded-xl ${groupType.iconClassName}`} aria-hidden="true"><Icon path={groupType.icon} size={1.1} /></span><h1 className="page-title">{groupinfo?.groupName}</h1></div>
                        <p className="page-copy">
                            Review balances, add expenses, and update members.
                        </p>
                    </div>
                    <div className="page-actions w-full sm:w-auto">
                        <Link
                            to={`/create_expense?g=${groupId}`}
                            className="ui-button ui-button-primary w-full sm:w-40"
                        >
                            Add Expense
                        </Link>
                        <Link to={`/group/${groupId}/edit`} className="ui-button ui-button-outline w-full sm:w-40">Edit Group</Link>
                        <button
                            className="ui-button ui-button-destructive w-full sm:w-40"
                            onClick={() => setSettleOpen(true)}
                        >
                            Settle Up
                        </button>
                    </div>
                </div>

                <div className="mb-4 flex items-stretch gap-2 md:mb-6">
                    <Link
                        to={`/group/${groupId}/edit#members`}
                        className="panel-card-soft group flex min-h-14 min-w-0 flex-1 items-center gap-3 rounded-2xl px-3 py-2 transition-colors hover:border-primary/30 hover:bg-primary/5 md:max-w-sm md:px-4"
                        aria-label={`Manage ${memberCount} ${
                            memberCount === 1 ? "member" : "members"
                        }`}
                    >
                        <span
                            className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary"
                            aria-hidden="true"
                        >
                            <Icon path={mdiAccountMultipleOutline} size={1} />
                        </span>
                        <span className="min-w-0 flex-1">
                            <span className="block text-sm font-semibold text-foreground">
                                {memberCount} {memberCount === 1 ? "member" : "members"}
                            </span>
                            <span className="block truncate text-xs text-foreground/60">
                                View and manage
                            </span>
                        </span>
                        <Icon
                            className="shrink-0 text-foreground/45 transition-transform group-hover:translate-x-0.5"
                            path={mdiChevronRight}
                            size={0.8}
                            aria-hidden="true"
                        />
                    </Link>
                    <button
                        type="button"
                        className="ui-button ui-button-destructive min-h-14 shrink-0 gap-1.5 px-3 text-xs md:hidden"
                        onClick={() => setSettleOpen(true)}
                    >
                        <Icon
                            path={mdiHandshakeOutline}
                            size={0.8}
                            aria-hidden="true"
                        />
                        <span>Settle up</span>
                    </button>
                </div>

                <div className="grid gap-4 md:gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.55fr)]">
                    <section className="panel-card order-1 self-start rounded-[1.5rem] p-4 sm:p-5 xl:order-2 xl:sticky xl:top-6 xl:rounded-[2rem] xl:p-6">
                        <div className="flex items-center justify-between gap-3">
                            <div>
                                <div className="section-label">Balances</div>
                                <p className="mt-1 text-xs text-foreground/60 xl:hidden">
                                    Your current group summary
                                </p>
                            </div>
                            {balanceEntries.length > 0 && (
                                <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary xl:hidden">
                                    {balanceEntries.length}
                                </span>
                            )}
                        </div>

                        {balanceEntries.length === 0 ? (
                            <div className="metric-card mt-3 rounded-2xl p-3 text-sm text-foreground/70 xl:mt-4 xl:p-4">
                                All balanced. No one owes anything.
                            </div>
                        ) : (
                            <>
                                <div
                                    className="mt-3 grid gap-2 xl:hidden"
                                    data-testid="mobile-balance-summary"
                                >
                                    {visibleMobileBalances.map((entry) => (
                                        <BalanceEntry
                                            key={entry.id}
                                            label={entry.label}
                                            amount={entry.balance}
                                            currency={balance?.currency ?? ""}
                                            tone={entry.tone}
                                            compact
                                        />
                                    ))}
                                </div>
                                {balanceEntries.length > 2 && (
                                    <button
                                        type="button"
                                        className="ui-button ui-button-ghost ui-button-sm mt-3 w-full xl:hidden"
                                        onClick={() =>
                                            setShowAllBalances((showAll) => !showAll)
                                        }
                                        aria-expanded={showAllBalances}
                                    >
                                        {showAllBalances
                                            ? "Show fewer balances"
                                            : `View all ${balanceEntries.length} balances`}
                                    </button>
                                )}
                                <div className="mt-4 hidden gap-3 xl:grid">
                                    {balanceEntries.map((entry) => (
                                        <BalanceEntry
                                            key={entry.id}
                                            label={entry.label}
                                            amount={entry.balance}
                                            currency={balance?.currency ?? ""}
                                            tone={entry.tone}
                                        />
                                    ))}
                                </div>
                            </>
                        )}
                    </section>

                    <section className="order-2 space-y-8 xl:order-1">
                        <div className="panel-card rounded-[2rem] p-6 md:p-8">
                            <div className="flex items-center justify-between gap-3">
                                <div className="section-label">Unsettled</div>
                                <label className="flex min-h-11 items-center gap-2 text-sm font-medium text-foreground/70">
                                    <span className="hidden sm:inline">Order</span>
                                    <select
                                        aria-label="Expense order"
                                        className="ui-select min-h-11 w-auto bg-background py-2 pl-3 pr-8 text-sm"
                                        value={expenseOrder}
                                        onChange={(event) =>
                                            setExpenseOrder(
                                                event.target.value === "oldest"
                                                    ? "oldest"
                                                    : "newest"
                                            )
                                        }
                                    >
                                        <option value="newest">New to old</option>
                                        <option value="oldest">Old to new</option>
                                    </select>
                                </label>
                            </div>
                            <div
                                className="mt-4 space-y-4"
                                id="unsettled-expenses"
                            >
                                {unsettledExpenses.length === 0 &&
                                !unsettledLoading ? (
                                    <div className="metric-card rounded-[1.5rem] p-6 text-sm text-foreground/70">
                                        No expenses yet.
                                    </div>
                                ) : (
                                    unsettledExpenses.map((exp: ExpenseData) => (
                                        <ExpenseCard
                                            key={exp.expenseId}
                                            {...exp}
                                        />
                                    ))
                                )}
                                {unsettledLoading && (
                                    <div className="flex justify-center py-2">
                                        <span className="ui-spinner ui-spinner-sm"></span>
                                    </div>
                                )}
                                {unsettledHasMore &&
                                    !unsettledLoading &&
                                    unsettledExpenses.length > 0 && (
                                        <div className="flex pt-2 sm:justify-end">
                                            <button
                                                className="ui-button ui-button-ghost w-full sm:w-auto"
                                                onClick={loadMoreUnsettledExpenses}
                                            >
                                                Load more unsettled expenses
                                            </button>
                                        </div>
                                    )}
                                {!unsettledHasMore &&
                                    !unsettledLoading &&
                                    unsettledExpenses.length > 0 && (
                                    <p className="pt-2 text-center text-sm text-foreground/60 sm:text-right">
                                        No more unsettled expenses
                                    </p>
                                )}
                            </div>
                        </div>

                        <div className="panel-card rounded-[2rem] p-6 md:p-8">
                            <div className="section-label">Settled</div>
                            {!showSettled ? (
                                <div className="mt-4">
                                    <button
                                        className="ui-button ui-button-ghost w-full sm:w-auto"
                                        onClick={() => {
                                            setShowSettled(true);
                                        }}
                                    >
                                        Load Settled Expenses
                                    </button>
                                </div>
                            ) : (
                                <div className="mt-4 space-y-4">
                                    {settledExpenses.length === 0 &&
                                    !settledLoading ? (
                                        <div className="metric-card rounded-[1.5rem] p-6 text-sm text-foreground/70">
                                            No settled expenses yet.
                                        </div>
                                    ) : (
                                        settledExpenses.map(
                                            (exp: ExpenseData) => (
                                                <ExpenseCard
                                                    key={exp.expenseId}
                                                    {...exp}
                                                />
                                            )
                                        )
                                    )}
                                    {settledLoading && (
                                        <div className="flex justify-center py-2">
                                            <span className="ui-spinner ui-spinner-sm"></span>
                                        </div>
                                    )}
                                    {settledHasMore &&
                                        !settledLoading &&
                                        settledExpenses.length > 0 && (
                                            <div className="flex pt-2 sm:justify-end">
                                                <button
                                                    className="ui-button ui-button-ghost w-full sm:w-auto"
                                                    onClick={loadMoreSettledExpenses}
                                                >
                                                    Load more settled expenses
                                                </button>
                                            </div>
                                        )}
                                    {!settledHasMore &&
                                        !settledLoading &&
                                        settledExpenses.length > 0 && (
                                            <p className="pt-2 text-center text-sm text-foreground/60 sm:text-right">
                                                No more settled expenses
                                            </p>
                                        )}
                                </div>
                            )}
                        </div>
                    </section>
                </div>
            </div>

            <Link
                to={`/create_expense?g=${groupId}`}
                className="ui-button ui-button-primary group-add-expense-fab"
                aria-label="Add expense"
            >
                <Icon path={mdiPlus} size={1.05} aria-hidden="true" />
                <span>Add expense</span>
            </Link>

            <AlertDialog open={settleOpen} onOpenChange={setSettleOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Settle all expenses?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This marks the current group balances as settled.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <Button
                            variant="destructive"
                            onClick={() => {
                                void handleSettle().finally(() =>
                                    setSettleOpen(false)
                                );
                            }}
                        >
                            Settle
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
};

const BalanceEntry = ({
    label,
    amount,
    currency,
    tone,
    compact = false,
}: {
    label: string;
    amount: string;
    currency: string;
    tone: string;
    compact?: boolean;
}) => (
    <div
        className={`metric-card rounded-2xl text-sm ${
            compact ? "flex items-center justify-between gap-3 p-3" : "p-4"
        }`}
    >
        <div className="min-w-0 font-semibold">{label}</div>
        <div
            className={`shrink-0 font-semibold ${
                compact ? "text-base" : "mt-1 text-lg"
            } ${tone}`}
        >
            ${amount} {currency}
        </div>
    </div>
);

export default function GroupDetail() {
    return (
        <GroupDetailProvider>
            <GroupDetailContent />
        </GroupDetailProvider>
    );
}
