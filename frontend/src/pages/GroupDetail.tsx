import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
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

const GroupDetailContent = () => {
    const {
        groupinfo,
        balance,
        unsettledExpenses,
        unsettledLoading,
        unsettledHasMore,
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
    const settledSentinelRef = useRef<HTMLDivElement | null>(null);
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
        const sentinel = settledSentinelRef.current;
        if (!sentinel) return;

        const observer = new IntersectionObserver(
            (entries) => {
                if (entries[0]?.isIntersecting) {
                    loadMoreSettledExpenses();
                }
            },
            { rootMargin: "200px" }
        );

        observer.observe(sentinel);
        return () => observer.disconnect();
    }, [showSettled, loadMoreSettledExpenses]);

    if (loading) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="ui-spinner ui-spinner-lg"></span>
            </div>
        );
    }

    return (
        <div className="page-shell">
            <div className="page-container">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group</div>
                        <h1 className="page-title">{groupinfo?.groupName}</h1>
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
                        <Link
                            to={`/add_member?g=${groupId}`}
                            className="ui-button ui-button-outline w-full sm:w-40"
                        >
                            Add Members
                        </Link>
                        <button
                            className="ui-button ui-button-destructive w-full sm:w-40"
                            onClick={() => setSettleOpen(true)}
                        >
                            Settle Up
                        </button>
                    </div>
                </div>

                <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.55fr)]">
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
                            <div className="section-label">Unsettled</div>
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
                                <div className="pt-2">
                                    <button
                                        className="ui-button ui-button-ghost w-full sm:w-auto"
                                        onClick={loadMoreUnsettledExpenses}
                                    >
                                        Load More
                                    </button>
                                </div>
                            )}
                            </div>
                        </div>

                        <div className="panel-card rounded-[2rem] p-6 md:p-8">
                            <div className="section-label">Settled</div>
                            {!showSettled ? (
                                <div className="mt-4">
                                <button
                                    className="ui-button ui-button-ghost w-full sm:w-auto"
                                    onClick={async () => {
                                        setShowSettled(true);
                                        await loadSettledExpenses();
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
                                    settledExpenses.map((exp: ExpenseData) => (
                                        <ExpenseCard
                                            key={exp.expenseId}
                                            {...exp}
                                        />
                                    ))
                                )}
                                {settledLoading && (
                                    <div className="flex justify-center py-2">
                                        <span className="ui-spinner ui-spinner-sm"></span>
                                    </div>
                                )}
                                {settledHasMore && (
                                    <div
                                        ref={settledSentinelRef}
                                        className="h-6"
                                    />
                                )}
                                </div>
                            )}
                        </div>
                    </section>
                </div>
            </div>

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
