import {
    createContext,
    useContext,
    FormEvent,
    ChangeEvent,
    Dispatch,
    SetStateAction,
} from "react";
import { Rule } from "../types/splitRule";
import { GroupListItem, GroupMember } from "../types/group";
import { ExpenseTypeItem } from "../types/expense";

export interface expenseFormData {
    groupId: string;
    expenseType: string;
    description: string;
    occurredOn: string;
    currency: string;
    total: number;
    splitRule: Rule;

    payerUserId: string;
    ledgers: {
        id: string;
        userId: string;
        share: number;
    }[];
}

export interface EditExpenseContextType {
    formData: expenseFormData;
    setFormData: Dispatch<SetStateAction<expenseFormData>>;
    groupList: GroupListItem[];
    expenseTypes: ExpenseTypeItem[];
    groupMembers: GroupMember[];
    indicatorShow: boolean;
    dataOk: boolean;
    hasChanges: boolean;
    ledgerShareOk: boolean;
    ledgerShareMessage: string;
    handleUpdateExpense: (e: FormEvent) => Promise<void>;
    handleFormDataChange: (
        e: ChangeEvent<HTMLSelectElement | HTMLInputElement>
    ) => void;
}

export const EditExpenseContext = createContext<
    EditExpenseContextType | undefined
>(undefined);

export const useEditExpense = () => {
    const context = useContext(EditExpenseContext);
    if (!context) {
        throw new Error(
            "useEditExpense must be used within a EditExpenseProvider"
        );
    }
    return context;
};
