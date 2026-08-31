import { useCallback, useRef, useState, useEffect, FormEvent, ChangeEvent, ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, getResponseError, ResponseError } from "../lib/api";
import { RegisterContext } from "../hooks/RegisterContextHooks";

export const RegisterProvider = ({ children }: { children: ReactNode }) => {
    const location = useLocation();
    const [token, setToken] = useState(() => location.hash.slice(1).trim());
    const navigate = useNavigate();

    const [formData, setFormData] = useState({
        nickname: "",
        firstname: "",
        lastname: "",
        email: "",
        password: "",
    });
    const [loading, setLoading] = useState(false);
    const [googleLoading, setGoogleLoading] = useState(false);
    const [validating, setValidating] = useState(true);
    const [error, setError] = useState("");
    const [tokenValid, setTokenValid] = useState(false);
    const [emailBound, setEmailBound] = useState(false);
    const googleSubmissionInFlight = useRef(false);

    useEffect(() => {
        if (!token) return;

        const safeURL = `${location.pathname}${location.search}`;
        navigate(safeURL, { replace: true });
    }, [location.pathname, location.search, navigate, token]);

    useEffect(() => {
        if (!token) {
            setValidating(false);
            return;
        }

        const exchangeInvitation = async () => {
            try {
                const response = await apiFetch(
                    "/register/invitation/exchange",
                    {
                        method: "POST",
                        body: JSON.stringify({ token }),
                    },
                    { authMode: "none" },
                );
                if (response.ok) {
                    const data = await response.json();
                    if (data.valid) {
                        const invitationEmail =
                            typeof data.email === "string"
                                ? data.email.trim()
                                : "";
                        setTokenValid(true);
                        setEmailBound(invitationEmail !== "");
                        setFormData((prev) => ({
                            ...prev,
                            email: invitationEmail,
                        }));
                        setToken("");
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

        exchangeInvitation();
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
                    }),
                },
                { authMode: "none" }
            );

            if (!response.ok) {
                const responseError = await getResponseError(
                    response,
                    "Registration failed",
                );
                throw new Error(registrationErrorMessage(responseError));
            }

            toast.success("Account created. You can now log in.");
            navigate("/login", { replace: true });
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

    const handleGoogleCredentialResponse = useCallback(
        async (response: GoogleCredentialResponse) => {
            if (googleSubmissionInFlight.current) return;
            if (!tokenValid || !response.credential?.trim()) {
                setError(
                    tokenValid
                        ? "Google registration did not return a credential. Please try again."
                        : "Registration requires a valid invitation link.",
                );
                return;
            }

            googleSubmissionInFlight.current = true;
            setGoogleLoading(true);
            setError("");
            try {
                const registrationResponse = await apiFetch(
                    "/auth/google/register",
                    {
                        method: "POST",
                        headers: {
                            Authorization: `Bearer ${response.credential}`,
                        },
                        body: JSON.stringify({}),
                    },
                    { authMode: "none" },
                );

                if (!registrationResponse.ok) {
                    const responseError = await getResponseError(
                        registrationResponse,
                        "Google registration failed",
                    );
                    throw new Error(registrationErrorMessage(responseError));
                }

                toast.success(
                    "Google account registered. Continue with Google to log in.",
                );
                navigate("/login", { replace: true });
            } catch (err) {
                setError(
                    err instanceof Error
                        ? err.message
                        : "An unexpected error occurred",
                );
            } finally {
                googleSubmissionInFlight.current = false;
                setGoogleLoading(false);
            }
        },
        [navigate, tokenValid],
    );

    return (
        <RegisterContext.Provider
            value={{
                formData,
                loading,
                googleLoading,
                validating,
                error,
                tokenValid,
                emailBound,
                sessionReady: tokenValid,
                handleChange,
                handleSubmit,
                handleGoogleCredentialResponse,
            }}
        >
            {children}
        </RegisterContext.Provider>
    );
};

function registrationErrorMessage(error: ResponseError) {
    switch (error.code) {
        case "INVITATION_REQUIRED":
        case "INVITATION_INVALID":
        case "INVITATION_EXPIRED":
        case "INVITATION_USED":
            return "This invitation is no longer valid. Ask an administrator for a new invitation link.";
        case "INVITATION_EMAIL_MISMATCH":
            return "Choose the Google account that matches the email address on this invitation.";
        case "ACCOUNT_CONFLICT":
            return "An account already exists for this email or Google identity. Log in instead, or connect accounts from Account Settings.";
        default:
            return error.message;
    }
}
