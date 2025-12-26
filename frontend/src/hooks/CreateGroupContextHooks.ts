import { createContext, useContext, FormEvent } from "react";

export interface CreateGroupContextType {
    groupName: string;
    setGroupName: (name: string) => void;
    description: string;
    setDescription: (desc: string) => void;
    currency: string;
    setCurrency: (curr: string) => void;
    indicator: boolean;
    feedback: string;
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
