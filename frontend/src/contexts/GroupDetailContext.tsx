import { useState, useEffect, ReactNode, useRef, useCallback } from "react";
import { useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, getResponseErrorMessage } from "../lib/api";
import { GroupInfo } from "../types/group";
import { BalanceData } from "../types/balance";
import { ExpenseData } from "../types/expense";
import { GroupDetailContext } from "../hooks/GroupDetailContextHooks";

const SETTLE_EXPENSES_FALLBACK = "Failed to settle expenses.";

export const GroupDetailProvider = ({ children }: { children: ReactNode }) => {
    const { id: groupId } = useParams();
    const [groupinfo, setGroupInfo] = useState<GroupInfo | null>(null);
    const [balance, setBalance] = useState<BalanceData | null>(null);
    const [loading, setLoading] = useState(true);
    const [unsettledExpenses, setUnsettledExpenses] = useState<ExpenseData[]>(
        []
    );
    const [unsettledPage, setUnsettledPage] = useState(0);
    const [unsettledHasMore, setUnsettledHasMore] = useState(false);
    const [unsettledLoading, setUnsettledLoading] = useState(false);
    const [settledExpenses, setSettledExpenses] = useState<ExpenseData[]>([]);
    const [settledPage, setSettledPage] = useState(0);
    const [settledHasMore, setSettledHasMore] = useState(false);
    const [settledLoading, setSettledLoading] = useState(false);
    const initialUnsettledLoadedRef = useRef(false);

    const refreshGroupSummary = useCallback(async () => {
        if (!groupId) return;
        setLoading(true);
        try {
            const [groupRes, balanceRes] = await Promise.all([
                apiFetch(`/group/${groupId}`),
                apiFetch(`/balance/${groupId}`),
            ]);

            if (groupRes.ok) setGroupInfo(await groupRes.json());
            if (balanceRes.ok) setBalance(await balanceRes.json());
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    }, [groupId]);

    useEffect(() => {
        void refreshGroupSummary();
    }, [refreshGroupSummary]);

    const fetchExpensePage = useCallback(async (page: number) => {
        if (!groupId) return [];
        const response = await apiFetch(`/expense_list/${groupId}/${page}`);
        if (!response.ok) return [];
        return (await response.json()) as ExpenseData[];
    }, [groupId]);

    const refreshExpenseLists = useCallback(async () => {
        setUnsettledLoading(true);
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(0);
            setUnsettledExpenses(data.filter((exp) => !exp.isSettled));
            setSettledExpenses(data.filter((exp) => exp.isSettled));
            setUnsettledPage(1);
            setSettledPage(1);
            setUnsettledHasMore(data.length > 0);
            setSettledHasMore(data.length > 0);
        } catch (error) {
            console.error(error);
            setUnsettledHasMore(false);
            setSettledHasMore(false);
        } finally {
            setUnsettledLoading(false);
            setSettledLoading(false);
        }
    }, [fetchExpensePage]);

    useEffect(() => {
        if (!groupId) return;

        if (initialUnsettledLoadedRef.current) return;
        initialUnsettledLoadedRef.current = true;
        void refreshExpenseLists();
    }, [groupId, refreshExpenseLists]);

    const loadMoreUnsettledExpenses = async () => {
        if (unsettledLoading || !unsettledHasMore) return;
        setUnsettledLoading(true);
        try {
            const data = await fetchExpensePage(unsettledPage);
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
            console.error(error);
            setUnsettledHasMore(false);
        } finally {
            setUnsettledLoading(false);
        }
    };

    const loadSettledExpenses = async () => {
        if (settledLoading) return;
        setSettledLoading(true);
        setSettledHasMore(true);
        setSettledPage(0);
        setSettledExpenses([]);
        try {
            const data = await fetchExpensePage(0);
            if (data.length === 0) {
                setSettledHasMore(false);
                return;
            }
            setSettledExpenses(data.filter((exp) => exp.isSettled));
            setSettledPage(1);
        } catch (error) {
            console.error(error);
            setSettledHasMore(false);
        } finally {
            setSettledLoading(false);
        }
    };

    const loadMoreSettledExpenses = async () => {
        if (settledLoading || !settledHasMore) return;
        setSettledLoading(true);
        try {
            const data = await fetchExpensePage(settledPage);
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
            console.error(error);
            setSettledHasMore(false);
        } finally {
            setSettledLoading(false);
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
