import { useState, useEffect, FormEvent, ChangeEvent, ReactNode } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { apiFetch, getResponseErrorMessage } from "../lib/api";
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
                const response = await apiFetch(
                    `/invitations/${token}`,
                    {},
                    { authMode: "none" }
                );
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
            const response = await apiFetch(
                "/register",
                {
                    method: "POST",
                    body: JSON.stringify({
                        ...formData,
                        token: token,
                    }),
                },
                { authMode: "none" }
            );

            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Registration failed"
                    )
                );
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
