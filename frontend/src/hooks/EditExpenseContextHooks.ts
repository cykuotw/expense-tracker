import {
    createContext,
    useContext,
    FormEvent,
    ChangeEvent,
    Dispatch,
    SetStateAction,
} from "react";
import { ExpenseAllocation } from "../types/allocation";
import { AllocationCalculation } from "../lib/expenseAllocation";
import {
    GroupListItem,
    GroupMember,
    GroupMembersLoadStatus,
} from "../types/group";
import { ExpenseTypeItem } from "../types/expense";
import { CurrencyMetadata } from "../lib/money";

export interface expenseFormData {
    groupId: string;
    expenseType: string;
    description: string;
    occurredOn: string;
    currency: string;
    total: string;
    allocation: ExpenseAllocation;

    payerUserId: string;
}

export interface EditExpenseContextType {
    formData: expenseFormData;
    amountDigits: number | null;
    currencies: CurrencyMetadata[];
    setFormData: Dispatch<SetStateAction<expenseFormData>>;
    groupList: GroupListItem[];
    expenseTypes: ExpenseTypeItem[];
    groupMembers: GroupMember[];
    currentUserId: string;
    groupMembersLoadStatus: GroupMembersLoadStatus;
    reloadGroupMembers: () => void;
    indicatorShow: boolean;
    submissionError: string | null;
    dataOk: boolean;
    hasChanges: boolean;
    allocationCalculation: AllocationCalculation;
    mainFormVisited: boolean;
    markMainFormVisited: () => void;
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
