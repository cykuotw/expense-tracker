import { useState, useEffect, FormEvent, ChangeEvent, ReactNode } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { API_URL } from "../configs/config";
import { RegisterContext } from "../hooks/RegisterContextHooks";

export const RegisterProvider = ({ children }: { children: ReactNode }) => {
    const [searchParams] = useSearchParams();
    const token = searchParams.get("token");
    const navigate = useNavigate();

    const [formData, setFormData] = useState({
        nickname: "",
        firstname: "",
        lastname: "",
        email: "",
        password: "",
    });
    const [loading, setLoading] = useState(false);
    const [validating, setValidating] = useState(true);
    const [error, setError] = useState("");
    const [tokenValid, setTokenValid] = useState(false);

    useEffect(() => {
        if (!token) {
            setValidating(false);
            return;
        }

        const validateToken = async () => {
            try {
                const response = await fetch(`${API_URL}/invitations/${token}`);
                if (response.ok) {
                    const data = await response.json();
                    if (data.valid) {
                        setTokenValid(true);
                        setFormData((prev) => ({ ...prev, email: data.email }));
                    }
                } else {
                    setError("Invalid or expired invitation link.");
                }
            } catch {
                setError("Failed to validate invitation.");
            } finally {
                setValidating(false);
            }
        };

        validateToken();
    }, [token]);

    const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
        setFormData({ ...formData, [e.target.name]: e.target.value });
    };

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError("");

        try {
            const response = await fetch(`${API_URL}/register`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    ...formData,
                    token: token,
                }),
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || "Registration failed");
            }

            navigate("/login");
        } catch (err) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError("An unexpected error occurred");
            }
        } finally {
            setLoading(false);
        }
    };

    return (
        <RegisterContext.Provider
            value={{
                formData,
                loading,
                validating,
                error,
                tokenValid,
                token,
                handleChange,
                handleSubmit,
            }}
        >
            {children}
        </RegisterContext.Provider>
    );
};
