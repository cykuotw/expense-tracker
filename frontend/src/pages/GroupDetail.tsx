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
    const settledSentinelRef = useRef<HTMLDivElement | null>(null);

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
                            className="ui-button ui-button-primary w-full sm:w-auto"
                        >
                            Add Expense
                        </Link>
                        <Link
                            to={`/add_member?g=${groupId}`}
                            className="ui-button ui-button-outline w-full sm:w-auto"
                        >
                            Add Members
                        </Link>
                        <button
                            className="ui-button ui-button-destructive w-full sm:w-auto"
                            onClick={() => setSettleOpen(true)}
                        >
                            Settle Up
                        </button>
                    </div>
                </div>

                <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.55fr)]">
                    <section className="panel-card order-2 self-start rounded-[2rem] p-6 md:p-8 xl:sticky xl:top-6">
                        <div className="section-label">Balances</div>
                        <div className="mt-4 grid gap-3">
                        {!balance?.balances || balance.balances.length === 0 ? (
                            <div className="metric-card rounded-[1.5rem] p-4 text-sm text-foreground/70">
                                All balanced. No one owes anything.
                            </div>
                        ) : (
                            balance.balances.map((b) => {
                                if (b.receiverUserId == balance.currentUser) {
                                    return (
                                        <div
                                            key={b.id}
                                            className="metric-card rounded-[1.5rem] p-4 text-sm"
                                        >
                                            <div className="font-semibold">
                                                {b.senderUsername} owes you
                                            </div>
                                            <div className="text-lg font-semibold text-success">
                                                ${b.balance} {balance.currency}
                                            </div>
                                        </div>
                                    );
                                }
                                if (b.senderUserId == balance.currentUser) {
                                    return (
                                        <div
                                            key={b.id}
                                            className="metric-card rounded-[1.5rem] p-4 text-sm"
                                        >
                                            <div className="font-semibold">
                                                You owe {b.receiverUsername}
                                            </div>
                                            <div className="text-lg font-semibold text-destructive">
                                                ${b.balance} {balance.currency}
                                            </div>
                                        </div>
                                    );
                                }
                            })
                        )}
                        </div>
                    </section>

                    <section className="order-1 space-y-8">
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

export default function GroupDetail() {
    return (
        <GroupDetailProvider>
            <GroupDetailContent />
        </GroupDetailProvider>
    );
}
