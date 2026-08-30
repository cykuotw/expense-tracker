import { Link } from "react-router-dom";
import { useState } from "react";
import GroupCard from "../components/group/GroupCard";
import { HomeProvider } from "../contexts/HomeContext";
import { useHome } from "../contexts/HomeContextHooks";

const HomeContent = () => {
    const { groupCards, loading } = useHome();
    const [showAllMobileBalances, setShowAllMobileBalances] = useState(false);

    if (loading) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="ui-spinner ui-spinner-lg"></span>
            </div>
        );
    }

    const hasGroups = groupCards.length > 0;
    const unsettledGroups = groupCards.filter(
        (group) => group.balanceStatus !== "settled",
    );
    const groupsWithBalances = unsettledGroups.length;
    const unsettledTotals = unsettledGroups.reduce<
        Record<string, { owed: number; owing: number }>
    >((totals, group) => {
        const amount = Number(group.balanceAmount);
        if (Number.isNaN(amount)) {
            return totals;
        }

        const currentTotals = totals[group.currency] ?? { owed: 0, owing: 0 };
        if (group.balanceStatus === "owed") {
            currentTotals.owed += amount;
        } else if (group.balanceStatus === "owing") {
            currentTotals.owing += amount;
        }

        totals[group.currency] = currentTotals;
        return totals;
    }, {});
    const unsettledTotalEntries = Object.entries(unsettledTotals).sort(
        ([currencyA], [currencyB]) => currencyA.localeCompare(currencyB),
    );
    const mobileBalanceDetails = unsettledTotalEntries.flatMap(
        ([currency, totals]) => {
            const precision = currency === "NTD" ? 0 : 2;
            return [
                ...(totals.owed > 0
                    ? [
                          {
                              key: `${currency}-owed`,
                              label: `You are owed ${totals.owed.toFixed(precision)} ${currency}`,
                              tone: "text-success",
                          },
                      ]
                    : []),
                ...(totals.owing > 0
                    ? [
                          {
                              key: `${currency}-owing`,
                              label: `You owe ${totals.owing.toFixed(precision)} ${currency}`,
                              tone: "text-destructive",
                          },
                      ]
                    : []),
            ];
        },
    );
    const visibleMobileBalanceDetails = showAllMobileBalances
        ? mobileBalanceDetails
        : mobileBalanceDetails.slice(0, 2);

    return (
        <div className="page-shell">
            <div className="page-container">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Groups</div>
                        <h1 className="page-title">Keep expenses organized</h1>
                        <p className="page-copy">
                            Open a group to review balances and expenses, or
                            create a new group to get started.
                        </p>
                    </div>
                    <div className="page-actions">
                        <div
                            className="rounded-3xl bg-background/70 px-4 py-3 text-sm text-foreground/70"
                            data-testid="mobile-home-summary"
                        >
                            <div className="hidden sm:block">
                                <span className="font-semibold text-foreground stat-number">
                                    {groupCards.length}
                                </span>{" "}
                                active group{groupCards.length === 1 ? "" : "s"}
                            </div>
                            {hasGroups && (
                                <>
                                <div className="flex flex-wrap gap-x-2 gap-y-1 text-base font-semibold sm:hidden">
                                    {visibleMobileBalanceDetails.length > 0 ? (
                                        visibleMobileBalanceDetails.map(
                                            (detail) => (
                                                <span
                                                    key={detail.key}
                                                    className={`rounded-xl px-2.5 py-1 ${detail.tone} ${
                                                        detail.tone === "text-success"
                                                            ? "bg-success/12"
                                                            : "bg-destructive/12"
                                                    }`}
                                                >
                                                    {detail.label}
                                                </span>
                                            ),
                                        )
                                    ) : (
                                        <span className="rounded-xl bg-success/12 px-2.5 py-1 text-success">
                                            All settled
                                        </span>
                                    )}
                                </div>
                                <div className="mt-2 text-xs text-foreground/60 sm:hidden">
                                    {groupCards.length} active group
                                    {groupCards.length === 1 ? "" : "s"} ·{" "}
                                    {groupsWithBalances} unsettled
                                </div>
                                {mobileBalanceDetails.length > 2 && (
                                    <button
                                        type="button"
                                        className="mt-2 text-xs font-semibold text-primary underline-offset-4 hover:underline sm:hidden"
                                        onClick={() =>
                                            setShowAllMobileBalances((showAll) =>
                                                !showAll,
                                            )
                                        }
                                        aria-expanded={showAllMobileBalances}
                                    >
                                        {showAllMobileBalances
                                            ? "Show fewer balances"
                                            : `View all ${mobileBalanceDetails.length} balances`}
                                    </button>
                                )}
                                </>
                            )}
                        </div>
                        <Link
                            to="/create_group"
                            className="ui-button ui-button-primary w-full sm:w-auto"
                        >
                            Create Group
                        </Link>
                    </div>
                </div>

                <section
                    className={`panel-card rounded-[2rem] p-6 md:p-8 ${
                        hasGroups ? "hidden sm:block" : ""
                    }`}
                    data-testid="home-summary-panel"
                >
                    {hasGroups ? (
                        <>
                            <div className="section-label">Summary</div>
                            <div className="mt-4 grid gap-4 lg:grid-cols-[0.7fr_1.3fr]">
                                <div className="metric-card rounded-[1.5rem] p-5">
                                    <div className="text-sm text-foreground/60">
                                        Unsettled groups
                                    </div>
                                    <div className="mt-3 text-3xl font-bold tracking-[-0.04em] text-primary stat-number">
                                        {groupsWithBalances}
                                    </div>
                                    <div className="mt-2 text-sm text-foreground/70">
                                        group
                                        {groupsWithBalances === 1
                                            ? ""
                                            : "s"}{" "}
                                        with open balances
                                    </div>
                                </div>
                                <div className="metric-card rounded-[1.5rem] p-5">
                                    <div className="text-sm text-foreground/60">
                                        Total open balances
                                    </div>
                                    {unsettledTotalEntries.length > 0 ? (
                                        <div className="mt-4 grid gap-2 sm:grid-cols-2">
                                            {unsettledTotalEntries.map(
                                                ([currency, totals]) => (
                                                    <div
                                                        key={currency}
                                                        className="rounded-2xl bg-background/80 px-4 py-3"
                                                    >
                                                        <div className="text-xs uppercase tracking-[0.16em] text-foreground/50">
                                                            {currency}
                                                        </div>
                                                        <div className="mt-3 space-y-2">
                                                            {totals.owed > 0 ? (
                                                                <div className="rounded-xl bg-success/12 px-3 py-2 text-success">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.14em]">
                                                                        You are owed
                                                                    </div>
                                                                    <div className="mt-1 text-base font-semibold">
                                                                        {totals.owed.toFixed(
                                                                            currency ===
                                                                                "NTD"
                                                                                ? 0
                                                                                : 2,
                                                                        )}{" "}
                                                                        {currency}
                                                                    </div>
                                                                </div>
                                                            ) : null}
                                                            {totals.owing > 0 ? (
                                                                <div className="rounded-xl bg-destructive/12 px-3 py-2 text-destructive">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.14em]">
                                                                        You owe
                                                                    </div>
                                                                    <div className="mt-1 text-base font-semibold">
                                                                        {totals.owing.toFixed(
                                                                            currency ===
                                                                                "NTD"
                                                                                ? 0
                                                                                : 2,
                                                                        )}{" "}
                                                                        {currency}
                                                                    </div>
                                                                </div>
                                                            ) : null}
                                                        </div>
                                                    </div>
                                                ),
                                            )}
                                        </div>
                                    ) : (
                                        <div className="mt-4 rounded-2xl bg-background/80 px-4 py-3 text-sm text-foreground/60">
                                            All groups are settled.
                                        </div>
                                    )}
                                </div>
                            </div>
                        </>
                    ) : (
                        <>
                            <div className="section-label">Get started</div>
                            <h2 className="mt-3 text-2xl font-semibold tracking-[-0.03em] text-foreground">
                                Create your first group
                            </h2>
                            <p className="mt-3 max-w-2xl text-sm leading-6 text-foreground/70">
                                Start a group to track shared expenses and
                                balances in one place.
                            </p>
                            <Link
                                to="/create_group"
                                className="ui-button ui-button-primary mt-6 w-full sm:w-auto"
                            >
                                Create Group
                            </Link>
                        </>
                    )}
                </section>

                {hasGroups ? (
                    <div
                        className="mt-6 grid gap-5 sm:mt-8 sm:grid-cols-2 xl:grid-cols-3"
                        data-testid="group-card-list"
                    >
                        {groupCards.map((group) => (
                            <GroupCard key={group.id} {...group} />
                        ))}
                    </div>
                ) : null}
            </div>
        </div>
    );
};

export default function Home() {
    return (
        <HomeProvider>
            <HomeContent />
        </HomeProvider>
    );
}
