import { createContext, useContext, FormEvent } from "react";
import { ExpenseDetailData } from "../types/expense";

export interface ExpenseDetailContextType {
    expenseDetail: ExpenseDetailData | null;
    formattedDate: string;
    expenseId: string;
    handleDeleteExpense: (e: FormEvent) => Promise<void>;
}

export const ExpenseDetailContext = createContext<
    ExpenseDetailContextType | undefined
>(undefined);

export const useExpenseDetail = () => {
    const context = useContext(ExpenseDetailContext);
    if (!context) {
        throw new Error(
            "useExpenseDetail must be used within a ExpenseDetailProvider"
        );
    }
    return context;
};
