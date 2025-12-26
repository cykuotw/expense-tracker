import {
    createContext,
    useContext,
    FormEvent,
    Dispatch,
    SetStateAction,
} from "react";

export interface Invitation {
    id: string;
    token: string;
    email: string;
    expiresAt: string;
    usedAt: string | null;
    createdAt: string;
}

export interface InviteUserContextType {
    email: string;
    setEmail: Dispatch<SetStateAction<string>>;
    token: string;
    error: string;
    loading: boolean;
    invitations: Invitation[];
    handleSubmit: (e: FormEvent) => Promise<void>;
    copyLink: (tokenToCopy: string) => void;
}

export const InviteUserContext = createContext<
    InviteUserContextType | undefined
>(undefined);

export const useInviteUser = () => {
    const context = useContext(InviteUserContext);
    if (!context) {
        throw new Error(
            "useInviteUser must be used within a InviteUserProvider"
        );
    }
    return context;
};
