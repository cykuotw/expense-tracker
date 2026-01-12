import { useState, useEffect, ReactNode, MouseEvent, useRef } from "react";
import { useParams } from "react-router-dom";
import { API_URL } from "../configs/config";
import { GroupInfo } from "../types/group";
import { BalanceData } from "../types/balance";
import { ExpenseData } from "../types/expense";
import { GroupDetailContext } from "../hooks/GroupDetailContextHooks";

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

    useEffect(() => {
        if (!groupId) return;

        const fetchData = async () => {
            setLoading(true);
            try {
                const [groupRes, balanceRes] = await Promise.all([
                    fetch(`${API_URL}/group/${groupId}`, {
                        credentials: "include",
                    }),
                    fetch(`${API_URL}/balance/${groupId}`, {
                        credentials: "include",
                    }),
                ]);

                if (groupRes.ok) setGroupInfo(await groupRes.json());
                if (balanceRes.ok) setBalance(await balanceRes.json());
            } catch (error) {
                console.error(error);
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, [groupId]);

    const fetchExpensePage = async (page: number) => {
        if (!groupId) return [];
        const response = await fetch(
            `${API_URL}/expense_list/${groupId}/${page}`,
            {
                credentials: "include",
            }
        );
        if (!response.ok) return [];
        return (await response.json()) as ExpenseData[];
    };

    useEffect(() => {
        if (!groupId) return;

        const loadInitialUnsettled = async () => {
            if (initialUnsettledLoadedRef.current) return;
            if (unsettledLoading) return;
            initialUnsettledLoadedRef.current = true;
            setUnsettledLoading(true);
            setUnsettledHasMore(true);
            setUnsettledPage(0);
            setUnsettledExpenses([]);
            try {
                const response = await fetch(
                    `${API_URL}/expense_list/${groupId}/0`,
                    {
                        credentials: "include",
                    }
                );
                if (!response.ok) {
                    setUnsettledHasMore(false);
                    return;
                }
                const data = (await response.json()) as ExpenseData[];
                if (data.length === 0) {
                    setUnsettledHasMore(false);
                    return;
                }
                setUnsettledExpenses(data.filter((exp) => !exp.isSettled));
                setUnsettledPage(1);
            } catch (error) {
                console.error(error);
                setUnsettledHasMore(false);
            } finally {
                setUnsettledLoading(false);
            }
        };

        loadInitialUnsettled();
    }, [groupId, unsettledLoading]);

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

    const handleSettle = async (e: MouseEvent<HTMLButtonElement>) => {
        e.preventDefault();
        if (!groupId) return;

        try {
            const response = await fetch(
                `${API_URL}/settle_expense/${groupId}`,
                {
                    method: "PUT",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            );
            if (response.ok) {
                console.log("Settlement successful");
            } else {
                console.error("Settlement failed");
            }
        } catch (error) {
            console.error("Error during settlement:", error);
        } finally {
            const dialog = document.getElementById(
                "settle_confirm"
            ) as HTMLDialogElement | null;
            dialog?.close();

            window.location.reload();
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
