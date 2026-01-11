import { useState, useEffect, ReactNode, FormEvent } from "react";
import { API_URL } from "../configs/config";
import { InviteUserContext, Invitation } from "../hooks/InviteUserContextHooks";

export const InviteUserProvider = ({ children }: { children: ReactNode }) => {
    const [token, setToken] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [invitations, setInvitations] = useState<Invitation[]>([]);

    const fetchInvitations = async () => {
        try {
            const response = await fetch(`${API_URL}/invitations`, {
                credentials: "include",
            });
            if (response.ok) {
                const data = await response.json();
                setInvitations(data);
            }
        } catch (err) {
            console.error("Failed to fetch invitations", err);
        }
    };

    useEffect(() => {
        fetchInvitations();
    }, []);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError("");
        setToken("");

        try {
            const response = await fetch(`${API_URL}/invitations`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                credentials: "include",
                body: JSON.stringify({}),
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || "Failed to create invitation");
            }

            const data = await response.json();
            setToken(data.token);
            fetchInvitations();
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

    const copyLink = (tokenToCopy: string) => {
        const link = `${window.location.origin}/register?token=${tokenToCopy}`;
        navigator.clipboard.writeText(link);
    };

    const expireInvitation = async (tokenToExpire: string) => {
        setError("");
        try {
            const response = await fetch(
                `${API_URL}/invitations/${tokenToExpire}/expire`,
                {
                    method: "POST",
                    credentials: "include",
                }
            );

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || "Failed to expire invitation");
            }

            fetchInvitations();
        } catch (err) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError("An unexpected error occurred");
            }
        }
    };

    return (
        <InviteUserContext.Provider
            value={{
                token,
                error,
                loading,
                invitations,
                handleSubmit,
                copyLink,
                expireInvitation,
            }}
        >
            {children}
        </InviteUserContext.Provider>
    );
};
