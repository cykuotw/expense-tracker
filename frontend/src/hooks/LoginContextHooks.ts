import { createContext, useContext, FormEvent } from "react";

export interface LoginContextType {
    email: string;
    password: string;
    feedback: string;
    loading: boolean;
    setEmail: (email: string) => void;
    setPassword: (password: string) => void;
    handleLoginSubmit: (e: FormEvent) => void;
}

export const LoginContext = createContext<LoginContextType | undefined>(
    undefined
);

export const useLogin = () => {
    const context = useContext(LoginContext);
    if (!context) {
        throw new Error("useLogin must be used within a LoginProvider");
    }
    return context;
};
