import {
    createContext,
    useContext,
    FormEvent,
    Dispatch,
    SetStateAction,
} from "react";
import { RelatedUser } from "../types/group";
import { UserData } from "../types/user";

export interface AddMemberContextType {
    groupId: string | null;
    loading: boolean;
    relatedUserList: RelatedUser[];
    email: string;
    setEmail: Dispatch<SetStateAction<string>>;
    newMember: UserData | null;
    handleSubmitRelatedUsers: (e: FormEvent) => Promise<void>;
    handleAddNewMember: () => void;
}

export const AddMemberContext = createContext<AddMemberContextType | undefined>(
    undefined
);

export const useAddMember = () => {
    const context = useContext(AddMemberContext);
    if (!context) {
        throw new Error("useAddMember must be used within a AddMemberProvider");
    }
    return context;
};
