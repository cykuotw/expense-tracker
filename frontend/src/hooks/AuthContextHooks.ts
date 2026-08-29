import { createContext, useContext } from "react";
import { UserRole } from "../types/role";

export interface AuthContextType {
    userID: string | null;
    isAuthenticated: boolean;
    role: UserRole | null;
    loading: boolean;
    isOffline: boolean;
    refreshSession: () => Promise<boolean>;
    markLoggedIn: () => Promise<boolean>;
    logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(
    undefined
);

export function useAuth() {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error("useAuth must be used within an AuthProvider");
    }
    return context;
}
