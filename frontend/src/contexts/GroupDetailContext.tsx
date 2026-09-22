import { useState, useEffect, ReactNode, useRef, useCallback } from "react";
import { useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { GroupInfo } from "../types/group";
import { BalanceData } from "../types/balance";
import { ExpenseData } from "../types/expense";
import {
    ExpenseListOrder,
    GroupDetailContext,
} from "../hooks/GroupDetailContextHooks";

const SETTLE_EXPENSES_FALLBACK = "Failed to settle expenses.";
const SETTLEMENT_ACCEPTED_UNCONFIRMED =
    "Settlement was accepted, but the latest state could not be confirmed. Refresh before trying again.";
const SETTLEMENT_STATUS_UNKNOWN =
    "Settlement status is unknown. Refresh before trying again.";
const SETTLEMENT_NOT_CONFIRMED =
    "Settlement was not confirmed. Review the latest balances before trying again.";
type ExpenseListStatus = "unsettled" | "settled";
type ExpenseListPage = {
    expenses: ExpenseData[];
    hasMore: boolean;
};
type GroupOverviewSnapshot = {
    group?: GroupInfo;
    balance?: BalanceData;
    expenses?: ExpenseListPage;
};

export const GroupDetailProvider = ({ children }: { children: ReactNode }) => {
    const { id: groupId } = useParams();
    const [groupinfo, setGroupInfo] = useState<GroupInfo | null>(null);
    const [balance, setBalance] = useState<BalanceData | null>(null);
    const [loading, setLoading] = useState(true);
    const [unsettledExpenses, setUnsettledExpenses] = useState<ExpenseData[]>(
        []
    );
    const [expenseOrder, setExpenseOrder] =
        useState<ExpenseListOrder>("newest");
    const [expenseListRefreshVersion, setExpenseListRefreshVersion] =
        useState(0);
    const [unsettledPage, setUnsettledPage] = useState(0);
    const [unsettledHasMore, setUnsettledHasMore] = useState(false);
    const [unsettledLoading, setUnsettledLoading] = useState(false);
    const [settledExpenses, setSettledExpenses] = useState<ExpenseData[]>([]);
    const [settledPage, setSettledPage] = useState(0);
    const [settledHasMore, setSettledHasMore] = useState(false);
    const [settledLoading, setSettledLoading] = useState(false);
    const [settlementPending, setSettlementPending] = useState(false);
    const unsettledExpenseListGenerationRef = useRef(0);
    const settledExpenseListGenerationRef = useRef(0);
    const settlementInFlightRef = useRef(false);

    const refreshGroupSummary = useCallback(async (
        signal?: AbortSignal,
    ): Promise<GroupOverviewSnapshot | null> => {
        if (!groupId) return null;
        setLoading(true);
        try {
            const path =
                `/group_overview/${groupId}/0?order=${expenseOrder}&status=unsettled`;
            const response = signal
                ? await apiFetch(path, { signal })
                : await apiFetch(path);
            if (!response.ok) return null;
            const data = (await response.json()) as GroupOverviewSnapshot;
            if (signal?.aborted) return null;
            if (data.group) {
                setGroupInfo({ ...data.group, members: asArray(data.group.members) });
            }
            if (data.balance) {
                setBalance({ ...data.balance, balances: asArray(data.balance.balances) });
            }
            if (data.expenses) {
                const expenses = asArray<ExpenseData>(data.expenses.expenses).map((expense) => ({
                    ...expense,
                    payerUserIds: asArray<string>(expense.payerUserIds),
                    payerUsernames: asArray<string>(expense.payerUsernames),
                }));
                setUnsettledExpenses(expenses);
                setUnsettledPage(1);
                setUnsettledHasMore(data.expenses.hasMore === true);
                setSettledExpenses([]);
                setSettledPage(0);
                setSettledHasMore(false);
                setExpenseListRefreshVersion((version) => version + 1);
            }
            return data;
        } catch (error) {
            if (signal?.aborted) return null;
            console.error(error);
            return null;
        } finally {
            if (!signal?.aborted) {
                setLoading(false);
            }
        }
    }, [expenseOrder, groupId]);

    useEffect(() => {
        const abortController = new AbortController();
        void refreshGroupSummary(abortController.signal);
        return () => abortController.abort();
    }, [refreshGroupSummary]);

    const fetchExpensePage = useCallback(async (
        page: number,
        order: ExpenseListOrder,
        status: ExpenseListStatus
    ): Promise<ExpenseListPage> => {
        if (!groupId) return { expenses: [], hasMore: false };
        const response = await apiFetch(
            `/expense_list/${groupId}/${page}?order=${order}&status=${status}`
        );
        if (!response.ok) return { expenses: [], hasMore: false };
        const data: unknown = await response.json();
        if (typeof data !== "object" || data === null || Array.isArray(data)) {
            return { expenses: [], hasMore: false };
        }
        const pageData = data as { expenses?: unknown; hasMore?: unknown };
        return {
            expenses: asArray<ExpenseData>(pageData.expenses).map((expense) => ({
                ...expense,
                payerUserIds: asArray<string>(expense.payerUserIds),
                payerUsernames: asArray<string>(expense.payerUsernames),
            })),
            hasMore: pageData.hasMore === true,
        };
    }, [groupId]);

    const loadMoreUnsettledExpenses = async () => {
        if (unsettledLoading || !unsettledHasMore) return;
        const generation = unsettledExpenseListGenerationRef.current;
        setUnsettledLoading(true);
        try {
            const data = await fetchExpensePage(
                unsettledPage,
                expenseOrder,
                "unsettled"
            );
            if (generation !== unsettledExpenseListGenerationRef.current) return;
            setUnsettledExpenses((prev) => [...prev, ...data.expenses]);
            setUnsettledPage((prev) => prev + 1);
            setUnsettledHasMore(data.hasMore);
        } catch (error) {
            if (generation !== unsettledExpenseListGenerationRef.current) return;
            console.error(error);
            setUnsettledHasMore(false);
        } finally {
            if (generation === unsettledExpenseListGenerationRef.current) {
                setUnsettledLoading(false);
            }
        }
    };

    const loadSettledExpenses = useCallback(async () => {
        if (settledLoading) return;
        const generation = ++settledExpenseListGenerationRef.current;
        setSettledLoading(true);
        setSettledPage(0);
        setSettledExpenses([]);
        try {
            const data = await fetchExpensePage(0, expenseOrder, "settled");
            if (generation !== settledExpenseListGenerationRef.current) return;
            setSettledExpenses(data.expenses);
            setSettledPage(1);
            setSettledHasMore(data.hasMore);
        } catch (error) {
            if (generation !== settledExpenseListGenerationRef.current) return;
            console.error(error);
            setSettledHasMore(false);
        } finally {
            if (generation === settledExpenseListGenerationRef.current) {
                setSettledLoading(false);
            }
        }
    }, [expenseOrder, fetchExpensePage, settledLoading]);

    const loadMoreSettledExpenses = async () => {
        if (settledLoading || !settledHasMore) return;
        const generation = settledExpenseListGenerationRef.current;
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(
                settledPage,
                expenseOrder,
                "settled"
            );
            if (generation !== settledExpenseListGenerationRef.current) return;
            setSettledExpenses((prev) => [...prev, ...data.expenses]);
            setSettledPage((prev) => prev + 1);
            setSettledHasMore(data.hasMore);
        } catch (error) {
            if (generation !== settledExpenseListGenerationRef.current) return;
            console.error(error);
            setSettledHasMore(false);
        } finally {
            if (generation === settledExpenseListGenerationRef.current) {
                setSettledLoading(false);
            }
        }
    };

    const handleSettle = async (): Promise<boolean> => {
        if (!groupId || settlementInFlightRef.current) return false;

        settlementInFlightRef.current = true;
        setSettlementPending(true);
        try {
            const response = await apiFetch(`/settle_expense/${groupId}`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                },
            });
            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        SETTLE_EXPENSES_FALLBACK
                    )
                );
                return false;
            }

            const snapshot = await refreshGroupSummary();
            if (!snapshot) {
                toast.error(SETTLEMENT_ACCEPTED_UNCONFIRMED);
                return false;
            }
            if (!isSettlementCommitted(snapshot)) {
                toast.error(SETTLEMENT_NOT_CONFIRMED);
                return false;
            }
            return true;
        } catch {
            const snapshot = await refreshGroupSummary();
            if (isSettlementCommitted(snapshot)) {
                toast.success("Settlement confirmed.");
                return true;
            }
            toast.error(
                snapshot ? SETTLEMENT_NOT_CONFIRMED : SETTLEMENT_STATUS_UNKNOWN
            );
            return false;
        } finally {
            settlementInFlightRef.current = false;
            setSettlementPending(false);
        }
    };

    return (
        <GroupDetailContext.Provider
            value={{
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
                settlementPending,
                groupId,
                handleSettle,
                loadMoreUnsettledExpenses,
                loadSettledExpenses,
                loadMoreSettledExpenses,
            }}
        >
            {children}
        </GroupDetailContext.Provider>
    );
};

function isSettlementCommitted(snapshot: GroupOverviewSnapshot | null): boolean {
    if (!snapshot?.balance || !snapshot.expenses) return false;
    return (
        asArray(snapshot.balance.balances).length === 0 &&
        asArray(snapshot.expenses.expenses).length === 0
    );
}
