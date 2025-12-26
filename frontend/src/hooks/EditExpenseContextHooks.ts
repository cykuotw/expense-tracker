import {
    createContext,
    useContext,
    FormEvent,
    ChangeEvent,
    ReactElement,
    Dispatch,
    SetStateAction,
} from "react";
import { Rule } from "../types/splitRule";
import { GroupListItem, GroupMember } from "../types/group";

export interface expenseFormData {
    groupId: string;
    expenseType: string;
    description: string;
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
    expTypeOptions: ReactElement[];
    groupMembers: GroupMember[];
    feedback: string;
    indicatorShow: boolean;
    dataOk: boolean;
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
