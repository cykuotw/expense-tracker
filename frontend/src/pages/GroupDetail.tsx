import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { ExpenseData } from "../types/expense";
import ExpenseCard from "../components/expense/ExpanceCard";
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
                <span className="loading loading-spinner loading-lg"></span>
            </div>
        );
    }

    return (
        <div className="page-shell">
            <div className="page-container">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group Overview</div>
                        <h1 className="page-title">{groupinfo?.groupName}</h1>
                        <p className="page-copy">
                            Track balances, add expenses, and settle up when you
                            are ready.
                        </p>
                    </div>
                    <div className="page-actions w-full sm:w-auto">
                        <Link
                            to={`/create_expense?g=${groupId}`}
                            className="btn btn-neutral w-full sm:w-auto"
                        >
                            Add Expense
                        </Link>
                        <Link
                            to={`/add_member?g=${groupId}`}
                            className="btn btn-outline w-full sm:w-auto"
                        >
                            Add Members
                        </Link>
                        <button
                            className="btn btn-error w-full sm:w-auto"
                            onClick={() => {
                                const dialog = document.getElementById(
                                    "settle_confirm"
                                ) as HTMLDialogElement | null;
                                dialog?.showModal();
                            }}
                        >
                            Settle Up
                        </button>
                    </div>
                </div>

                <div className="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
                    <section className="panel-card rounded-[2rem] p-6 md:p-8">
                        <div className="section-label">Balances</div>
                        <div className="mt-4 grid gap-3">
                        {!balance?.balances || balance.balances.length === 0 ? (
                            <div className="metric-card rounded-[1.5rem] p-4 text-sm text-base-content/70">
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
                                            <div className="text-lg font-semibold text-error">
                                                ${b.balance} {balance.currency}
                                            </div>
                                        </div>
                                    );
                                }
                            })
                        )}
                        </div>
                    </section>

                    <section className="space-y-8">
                        <div className="panel-card rounded-[2rem] p-6 md:p-8">
                            <div className="section-label">Unsettled</div>
                            <div
                                className="mt-4 space-y-4"
                                id="unsettled-expenses"
                            >
                            {unsettledExpenses.length === 0 &&
                            !unsettledLoading ? (
                                <div className="metric-card rounded-[1.5rem] p-6 text-sm text-base-content/70">
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
                                    <span className="loading loading-spinner loading-md"></span>
                                </div>
                            )}
                            {unsettledHasMore &&
                                !unsettledLoading &&
                                unsettledExpenses.length > 0 && (
                                <div className="pt-2">
                                    <button
                                        className="btn btn-ghost w-full sm:w-auto"
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
                                    className="btn btn-ghost w-full sm:w-auto"
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
                                    <div className="metric-card rounded-[1.5rem] p-6 text-sm text-base-content/70">
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
                                        <span className="loading loading-spinner loading-md"></span>
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

            <dialog id="settle_confirm" className="modal">
                <div className="modal-box">
                    <h3 className="text-lg font-bold">Are you sure?</h3>
                    <p className="py-4">Press settle to settle expenses.</p>
                    <div className="modal-action">
                        <form method="dialog" className="flex space-x-1">
                            <button
                                className="btn btn-outline btn-error w-1/2"
                                onClick={handleSettle}
                            >
                                Settle
                            </button>
                            <button className="btn w-1/2">Cancel</button>
                        </form>
                    </div>
                </div>
                <form method="dialog" className="modal-backdrop">
                    <button>close</button>
                </form>
            </dialog>
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
