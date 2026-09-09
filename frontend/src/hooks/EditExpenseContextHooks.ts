import {
    createContext,
    useContext,
    FormEvent,
    ChangeEvent,
    Dispatch,
    SetStateAction,
} from "react";
import { Rule } from "../types/splitRule";
import {
    GroupListItem,
    GroupMember,
    GroupMembersLoadStatus,
} from "../types/group";
import { ExpenseTypeItem } from "../types/expense";

export interface expenseFormData {
    groupId: string;
    expenseType: string;
    description: string;
    occurredOn: string;
    currency: string;
    total: string;
    splitRule: Rule;

    payerUserId: string;
    ledgers: {
        id: string;
        userId: string;
        share: string;
    }[];
}

export interface EditExpenseContextType {
    formData: expenseFormData;
    amountDigits: number | null;
    setFormData: Dispatch<SetStateAction<expenseFormData>>;
    groupList: GroupListItem[];
    expenseTypes: ExpenseTypeItem[];
    groupMembers: GroupMember[];
    groupMembersLoadStatus: GroupMembersLoadStatus;
    reloadGroupMembers: () => void;
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
