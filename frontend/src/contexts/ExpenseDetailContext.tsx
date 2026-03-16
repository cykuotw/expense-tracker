import { useState, useEffect, useMemo, ReactNode, FormEvent } from "react";
import { useParams } from "react-router-dom";
import { apiFetch } from "../lib/api";
import { ExpenseDetailData } from "../types/expense";
import { ExpenseDetailContext } from "../hooks/ExpenseDetailContextHooks";

export const ExpenseDetailProvider = ({
    children,
}: {
    children: ReactNode;
}) => {
    const { id: expenseId = "" } = useParams();
    const [expenseDetail, setExpenseDetail] =
        useState<ExpenseDetailData | null>(null);

    const formattedDate = useMemo(() => {
        if (!expenseDetail?.expenseTime) return "";
        return new Date(expenseDetail.expenseTime).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }, [expenseDetail?.expenseTime]);

    useEffect(() => {
        if (!expenseId) return;

        const fetchExpenseDetail = async () => {
            try {
                const response = await apiFetch(`/expense/${expenseId}`, {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const data: ExpenseDetailData = await response.json();
                setExpenseDetail(data);
            } catch (error) {
                console.log(error);
            }
        };

        fetchExpenseDetail();
    }, [expenseId]);

    const handleDeleteExpense = async (e: FormEvent) => {
        e.preventDefault();
        if (!expenseDetail?.groupId || !expenseId) return;

        try {
            const response = await apiFetch(`/delete_expense/${expenseId}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
            });

            if (response.status === 200) {
                window.location.href = `/group/${expenseDetail.groupId}`;
            }
        } catch (error) {
            console.log(error);
        }
    };

    return (
        <ExpenseDetailContext.Provider
            value={{
                expenseDetail,
                formattedDate,
                expenseId,
                handleDeleteExpense,
            }}
        >
            {children}
        </ExpenseDetailContext.Provider>
    );
};
