import { useState, useEffect, ReactNode, FormEvent } from "react";
import { toast } from "react-hot-toast";
import { API_URL } from "../configs/config";
import { InviteUserContext, Invitation } from "../hooks/InviteUserContextHooks";

export const InviteUserProvider = ({ children }: { children: ReactNode }) => {
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

            toast.success("Invitation created", { duration: 1000 });
            fetchInvitations();
        } catch (err) {
            if (err instanceof Error) {
                toast.error(err.message);
            } else {
                toast.error("An unexpected error occurred");
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
            toast.success("Invitation expired", { duration: 1000 });
        } catch (err) {
            if (err instanceof Error) {
                toast.error(err.message);
            } else {
                toast.error("An unexpected error occurred");
            }
        }
    };

    return (
        <InviteUserContext.Provider
            value={{
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
