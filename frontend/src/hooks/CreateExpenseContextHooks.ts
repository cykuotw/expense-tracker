import {
    createContext,
    useContext,
    FormEvent,
    ReactElement,
    Dispatch,
    SetStateAction,
} from "react";
import { Rule } from "../types/splitRule";
import { GroupListItem, GroupMember } from "../types/group";

export interface CreateExpenseContextType {
    groupId: string | null;
    selectedGroupId: string | null;
    setSelectedGroupId: Dispatch<SetStateAction<string | null>>;
    selectedExpenseTypeId: string;
    setSelectedExpenseTypeId: Dispatch<SetStateAction<string>>;
    total: number;
    setTotal: Dispatch<SetStateAction<number>>;
    description: string;
    setDescription: Dispatch<SetStateAction<string>>;
    currency: string;
    setCurrency: Dispatch<SetStateAction<string>>;
    payer: string;
    setPayer: Dispatch<SetStateAction<string>>;
    selectedRule: Rule;
    setSelectedRule: Dispatch<SetStateAction<Rule>>;
    ledgers: { userId: string; share: number }[];
    setLedgers: Dispatch<SetStateAction<{ userId: string; share: number }[]>>;

    feedback: string;
    indicatorShow: boolean;
    dataOk: boolean;
    ledgerShareOk: boolean;
    ledgerShareMessage: string;

    groupList: GroupListItem[];
    expTypeOptions: ReactElement[];
    groupMembers: GroupMember[];

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
