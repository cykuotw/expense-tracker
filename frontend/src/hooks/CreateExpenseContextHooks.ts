import {
    createContext,
    useContext,
    FormEvent,
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
    payer: string;
    setPayer: Dispatch<SetStateAction<string>>;
    allocation: ExpenseAllocation;
    setAllocation: Dispatch<SetStateAction<ExpenseAllocation>>;
    allocationCalculation: AllocationCalculation;
    mainFormVisited: boolean;
    markMainFormVisited: () => void;

    indicatorShow: boolean;
    dataOk: boolean;

    groupList: GroupListItem[];
    expenseTypes: ExpenseTypeItem[];
    groupMembers: GroupMember[];
    currentUserId: string;
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
