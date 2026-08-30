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
    const [unsettledPage, setUnsettledPage] = useState(0);
    const [unsettledHasMore, setUnsettledHasMore] = useState(false);
    const [unsettledLoading, setUnsettledLoading] = useState(false);
    const [settledExpenses, setSettledExpenses] = useState<ExpenseData[]>([]);
    const [settledPage, setSettledPage] = useState(0);
    const [settledHasMore, setSettledHasMore] = useState(false);
    const [settledLoading, setSettledLoading] = useState(false);
    const expenseListGenerationRef = useRef(0);

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
        order: ExpenseListOrder
    ) => {
        if (!groupId) return [];
        const response = await apiFetch(
            `/expense_list/${groupId}/${page}?order=${order}`
        );
        if (!response.ok) return [];
        const expenses = asArray<ExpenseData>(await response.json());
        return expenses.map((expense) => ({
            ...expense,
            payerUserIds: asArray<string>(expense.payerUserIds),
            payerUsernames: asArray<string>(expense.payerUsernames),
        }));
    }, [groupId]);

    const refreshExpenseLists = useCallback(async () => {
        const generation = ++expenseListGenerationRef.current;
        setUnsettledLoading(true);
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(0, expenseOrder);
            if (generation !== expenseListGenerationRef.current) return;
            setUnsettledExpenses(data.filter((exp) => !exp.isSettled));
            setSettledExpenses(data.filter((exp) => exp.isSettled));
            setUnsettledPage(1);
            setSettledPage(1);
            setUnsettledHasMore(data.length > 0);
            setSettledHasMore(data.length > 0);
        } catch (error) {
            if (generation !== expenseListGenerationRef.current) return;
            console.error(error);
            setUnsettledHasMore(false);
            setSettledHasMore(false);
        } finally {
            if (generation === expenseListGenerationRef.current) {
                setUnsettledLoading(false);
                setSettledLoading(false);
            }
        }
    }, [expenseOrder, fetchExpensePage]);

    useEffect(() => {
        if (!groupId) return;

        void refreshExpenseLists();
    }, [groupId, refreshExpenseLists]);

    const loadMoreUnsettledExpenses = async () => {
        if (unsettledLoading || !unsettledHasMore) return;
        const generation = expenseListGenerationRef.current;
        setUnsettledLoading(true);
        try {
            const data = await fetchExpensePage(unsettledPage, expenseOrder);
            if (generation !== expenseListGenerationRef.current) return;
            if (data.length === 0) {
                setUnsettledHasMore(false);
                return;
            }
            setUnsettledExpenses((prev) => [
                ...prev,
                ...data.filter((exp) => !exp.isSettled),
            ]);
            setUnsettledPage((prev) => prev + 1);
        } catch (error) {
            if (generation !== expenseListGenerationRef.current) return;
            console.error(error);
            setUnsettledHasMore(false);
        } finally {
            if (generation === expenseListGenerationRef.current) {
                setUnsettledLoading(false);
            }
        }
    };

    const loadSettledExpenses = async () => {
        if (settledLoading) return;
        const generation = ++expenseListGenerationRef.current;
        setSettledLoading(true);
        setSettledHasMore(true);
        setSettledPage(0);
        setSettledExpenses([]);
        try {
            const data = await fetchExpensePage(0, expenseOrder);
            if (generation !== expenseListGenerationRef.current) return;
            if (data.length === 0) {
                setSettledHasMore(false);
                return;
            }
            setSettledExpenses(data.filter((exp) => exp.isSettled));
            setSettledPage(1);
        } catch (error) {
            if (generation !== expenseListGenerationRef.current) return;
            console.error(error);
            setSettledHasMore(false);
        } finally {
            if (generation === expenseListGenerationRef.current) {
                setSettledLoading(false);
            }
        }
    };

    const loadMoreSettledExpenses = async () => {
        if (settledLoading || !settledHasMore) return;
        const generation = expenseListGenerationRef.current;
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(settledPage, expenseOrder);
            if (generation !== expenseListGenerationRef.current) return;
            if (data.length === 0) {
                setSettledHasMore(false);
                return;
            }
            setSettledExpenses((prev) => [
                ...prev,
                ...data.filter((exp) => exp.isSettled),
            ]);
            setSettledPage((prev) => prev + 1);
        } catch (error) {
            if (generation !== expenseListGenerationRef.current) return;
            console.error(error);
            setSettledHasMore(false);
        } finally {
            if (generation === expenseListGenerationRef.current) {
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
