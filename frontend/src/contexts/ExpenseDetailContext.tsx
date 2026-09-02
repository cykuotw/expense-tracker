import { useState, useEffect, useMemo, ReactNode, FormEvent } from "react";
import { useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { ExpenseDetailData } from "../types/expense";
import { ExpenseDetailContext } from "../hooks/ExpenseDetailContextHooks";
import { formatDateOnlyLong } from "../lib/dateOnly";

const DELETE_EXPENSE_FALLBACK = "Failed to delete expense.";
const LOAD_EXPENSE_FALLBACK = "Unable to load this expense.";

export const ExpenseDetailProvider = ({
    children,
}: {
    children: ReactNode;
}) => {
    const { id: expenseId = "" } = useParams();
    const [expenseDetail, setExpenseDetail] =
        useState<ExpenseDetailData | null>(null);
    const [loading, setLoading] = useState(Boolean(expenseId));
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const formattedDate = useMemo(() => {
        if (!expenseDetail?.occurredOn) return "";
        return formatDateOnlyLong(expenseDetail.occurredOn);
    }, [expenseDetail?.occurredOn]);

    useEffect(() => {
        if (!expenseId) {
            setLoading(false);
            return;
        }

        let active = true;

        const fetchExpenseDetail = async () => {
            setLoading(true);
            setErrorMessage(null);
            try {
                const response = await apiFetch(`/expense/${expenseId}`, {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                if (!response.ok) {
                    const message = await getResponseErrorMessage(
                        response,
                        LOAD_EXPENSE_FALLBACK
                    );
                    if (active) {
                        setErrorMessage(message);
                    }
                    return;
                }
                const responseData = (await response.json()) as ExpenseDetailData;
                if (active) {
                    setExpenseDetail({
                        ...responseData,
                        items: asArray<ExpenseDetailData["items"][number]>(
                            responseData.items
                        ),
                        ledgers: asArray<ExpenseDetailData["ledgers"][number]>(
                            responseData.ledgers
                        ),
                    });
                }
            } catch {
                if (active) {
                    setErrorMessage(LOAD_EXPENSE_FALLBACK);
                }
            } finally {
                if (active) {
                    setLoading(false);
                }
            }
        };

        fetchExpenseDetail();
        return () => {
            active = false;
        };
    }, [expenseId]);

    const handleDeleteExpense = async (e?: FormEvent) => {
        e?.preventDefault();
        if (!expenseDetail?.groupId || !expenseId) return;

        try {
            const response = await apiFetch(`/delete_expense/${expenseId}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
            });

            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        DELETE_EXPENSE_FALLBACK
                    )
                );
                return;
            }

            window.location.href = `/group/${expenseDetail.groupId}`;
        } catch {
            toast.error(DELETE_EXPENSE_FALLBACK);
        }
    };

    return (
        <ExpenseDetailContext.Provider
            value={{
                expenseDetail,
                formattedDate,
                expenseId,
                loading,
                errorMessage,
                handleDeleteExpense,
            }}
        >
            {children}
        </ExpenseDetailContext.Provider>
    );
};
