import { useState, useEffect, ReactNode, MouseEvent } from "react";
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
    const [expenseList, setExpenseList] = useState<ExpenseData[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!groupId) return;

        const fetchData = async () => {
            setLoading(true);
            try {
                const [groupRes, balanceRes, expenseRes] = await Promise.all([
                    fetch(`${API_URL}/group/${groupId}`, {
                        credentials: "include",
                    }),
                    fetch(`${API_URL}/balance/${groupId}`, {
                        credentials: "include",
                    }),
                    fetch(`${API_URL}/expense_list/${groupId}`, {
                        credentials: "include",
                    }),
                ]);

                if (groupRes.ok) setGroupInfo(await groupRes.json());
                if (balanceRes.ok) setBalance(await balanceRes.json());
                if (expenseRes.ok) setExpenseList(await expenseRes.json());
            } catch (error) {
                console.error(error);
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, [groupId]);

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
                expenseList,
                loading,
                groupId,
                handleSettle,
            }}
        >
            {children}
        </GroupDetailContext.Provider>
    );
};
