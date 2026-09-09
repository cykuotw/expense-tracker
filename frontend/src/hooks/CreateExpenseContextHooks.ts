import {
    createContext,
    useContext,
    FormEvent,
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

export interface CreateExpenseContextType {
    groupId: string | null;
    selectedGroupId: string | null;
    setSelectedGroupId: Dispatch<SetStateAction<string | null>>;
    selectedExpenseTypeId: string;
    setSelectedExpenseTypeId: Dispatch<SetStateAction<string>>;
    total: string;
    totalInput: string;
    setTotalInput: Dispatch<SetStateAction<string>>;
    description: string;
    setDescription: Dispatch<SetStateAction<string>>;
    occurredOn: string;
    setOccurredOn: Dispatch<SetStateAction<string>>;
    currency: string;
    amountDigits: number | null;
    setCurrency: Dispatch<SetStateAction<string>>;
    payer: string;
    setPayer: Dispatch<SetStateAction<string>>;
    selectedRule: Rule;
    setSelectedRule: Dispatch<SetStateAction<Rule>>;
    ledgers: { userId: string; share: string }[];
    setLedgers: Dispatch<SetStateAction<{ userId: string; share: string }[]>>;

    indicatorShow: boolean;
    dataOk: boolean;
    ledgerShareOk: boolean;
    ledgerShareMessage: string;

    groupList: GroupListItem[];
    expenseTypes: ExpenseTypeItem[];
    groupMembers: GroupMember[];
    groupMembersLoadStatus: GroupMembersLoadStatus;
    reloadGroupMembers: () => void;

    handleCreateExpense: (e: FormEvent) => Promise<void>;
}

export const CreateExpenseContext = createContext<
    CreateExpenseContextType | undefined
>(undefined);

export const useCreateExpense = () => {
    const context = useContext(CreateExpenseContext);
    if (!context) {
        throw new Error(
            "useCreateExpense must be used within a CreateExpenseProvider"
        );
    }
    return context;
};
