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
type ExpenseListStatus = "unsettled" | "settled";
type ExpenseListPage = {
    expenses: ExpenseData[];
    hasMore: boolean;
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
    const unsettledExpenseListGenerationRef = useRef(0);
    const settledExpenseListGenerationRef = useRef(0);

    const refreshGroupSummary = useCallback(async () => {
        if (!groupId) return;
        setLoading(true);
        try {
            const [groupRes, balanceRes] = await Promise.all([
                apiFetch(`/group/${groupId}`),
                apiFetch(`/balance/${groupId}`),
            ]);

            if (groupRes.ok) {
                const group = (await groupRes.json()) as GroupInfo;
                setGroupInfo({ ...group, members: asArray(group.members) });
            }
            if (balanceRes.ok) {
                const balance = (await balanceRes.json()) as BalanceData;
                setBalance({ ...balance, balances: asArray(balance.balances) });
            }
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    }, [groupId]);

    useEffect(() => {
        void refreshGroupSummary();
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

    const refreshExpenseLists = useCallback(async () => {
        const generation = ++unsettledExpenseListGenerationRef.current;
        ++settledExpenseListGenerationRef.current;
        setUnsettledLoading(true);
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(0, expenseOrder, "unsettled");
            if (generation !== unsettledExpenseListGenerationRef.current) return;
            setUnsettledExpenses(data.expenses);
            setUnsettledPage(1);
            setUnsettledHasMore(data.hasMore);
            setSettledExpenses([]);
            setSettledPage(0);
            setSettledHasMore(false);
        } catch (error) {
            if (generation !== unsettledExpenseListGenerationRef.current) return;
            console.error(error);
            setUnsettledHasMore(false);
            setSettledHasMore(false);
        } finally {
            if (generation === unsettledExpenseListGenerationRef.current) {
                setUnsettledLoading(false);
                setSettledLoading(false);
                setExpenseListRefreshVersion((version) => version + 1);
            }
        }
    }, [expenseOrder, fetchExpensePage]);

    useEffect(() => {
        if (!groupId) return;

        void refreshExpenseLists();
    }, [groupId, refreshExpenseLists]);

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

    const handleSettle = async () => {
        if (!groupId) return;

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
                return;
            }

            await Promise.all([refreshGroupSummary(), refreshExpenseLists()]);
        } catch {
            toast.error(SETTLE_EXPENSES_FALLBACK);
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
