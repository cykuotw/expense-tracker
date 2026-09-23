import { createContext, useContext, FormEvent } from "react";
import { GroupCurrencySettings } from "../types/group";

export interface CreateGroupContextType {
    groupName: string;
    setGroupName: (name: string) => void;
    description: string;
    setDescription: (desc: string) => void;
    currency: string;
    setCurrency: (curr: string) => void;
    currencySettings: GroupCurrencySettings;
    setCurrencySettings: (settings: GroupCurrencySettings) => void;
    groupType: string;
    setGroupType: (type: string) => void;
    createdGroupId: string | null;
    indicator: boolean;
    dataOk: boolean;
    createGroup: (e: FormEvent) => Promise<void>;
}

export const CreateGroupContext = createContext<
    CreateGroupContextType | undefined
>(undefined);

export const useCreateGroup = () => {
    const context = useContext(CreateGroupContext);
    if (!context) {
        throw new Error(
            "useCreateGroup must be used within a CreateGroupProvider"
        );
    }
    return context;
};
