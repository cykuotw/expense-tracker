import { createContext, useContext, FormEvent, ChangeEvent } from "react";

export interface RegisterContextType {
    formData: {
        nickname: string;
        firstname: string;
        lastname: string;
        email: string;
        password: string;
    };
    loading: boolean;
    validating: boolean;
    error: string;
    tokenValid: boolean;
    token: string | null;
    handleChange: (e: ChangeEvent<HTMLInputElement>) => void;
    handleSubmit: (e: FormEvent) => void;
}

export const RegisterContext = createContext<RegisterContextType | undefined>(
    undefined
);

export const useRegister = () => {
    const context = useContext(RegisterContext);
    if (!context) {
        throw new Error("useRegister must be used within a RegisterProvider");
    }
    return context;
};
