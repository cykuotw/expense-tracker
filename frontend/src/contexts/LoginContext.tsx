import { useState, ReactNode, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { API_URL } from "../configs/config";
import { LoginContext } from "../hooks/LoginContextHooks";

export const LoginProvider = ({ children }: { children: ReactNode }) => {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [feedback, setFeedback] = useState("");
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();

    const handleLoginSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setFeedback("");

        try {
            const response = await fetch(`${API_URL}/login`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email: email, password: password }),
                credentials: "include",
            });

            const data = await response.json();
            if (!response.ok) throw new Error(data.message || "Login failed");

            setFeedback("✅ Login successful!");

            setTimeout(() => {
                navigate("/");
            }, 500);
        } catch (error) {
            setFeedback(`❌ ${(error as Error).message}`);
        } finally {
            setLoading(false);
        }
    };

    return (
        <LoginContext.Provider
            value={{
                email,
                password,
                feedback,
                loading,
                setEmail,
                setPassword,
                handleLoginSubmit,
            }}
        >
            {children}
        </LoginContext.Provider>
    );
};
