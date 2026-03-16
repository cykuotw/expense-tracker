import { useState, ReactNode, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/AuthContextHooks";
import { LoginContext } from "../hooks/LoginContextHooks";
import { apiFetch, getResponseErrorMessage } from "../lib/api";

export const LoginProvider = ({ children }: { children: ReactNode }) => {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [feedback, setFeedback] = useState("");
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();
    const { markLoggedIn } = useAuth();

    const handleLoginSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setFeedback("");

        try {
            const response = await apiFetch(
                "/login",
                {
                    method: "POST",
                    body: JSON.stringify({ email: email, password: password }),
                },
                { authMode: "none" }
            );

            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(response, "Login failed")
                );
            }

            await markLoggedIn();

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
