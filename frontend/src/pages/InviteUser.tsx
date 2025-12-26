import { useState, useEffect, FormEvent } from "react";
import { API_URL } from "../configs/config";

interface Invitation {
    id: string;
    token: string;
    email: string;
    expiresAt: string;
    usedAt: string | null;
    createdAt: string;
}

const InviteUser = () => {
    const [email, setEmail] = useState("");
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
                body: JSON.stringify({ email }),
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

    return (
        <div className="flex flex-col items-center mt-10 gap-10">
            <div className="card w-96 bg-base-100 shadow-xl border border-base-200">
                <div className="card-body">
                    <h2 className="card-title justify-center mb-4">
                        Invite User
                    </h2>
                    <form onSubmit={handleSubmit}>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">
                                    Email Address
                                </span>
                            </label>
                            <input
                                type="email"
                                placeholder="email@example.com"
                                className="input input-bordered w-full"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                required
                            />
                        </div>

                        {error && (
                            <div className="alert alert-error mt-4 text-sm">
                                <span>{error}</span>
                            </div>
                        )}

                        {token && (
                            <div className="alert alert-success mt-4 text-sm flex-col items-start">
                                <span className="font-bold">
                                    Invitation Created!
                                </span>
                                <div className="break-all mt-1">
                                    Token: {token}
                                </div>
                                <div className="break-all mt-1 text-xs opacity-75">
                                    Link: {window.location.origin}
                                    /register?token={token}
                                </div>
                            </div>
                        )}

                        <div className="card-actions justify-end mt-6">
                            <button
                                type="submit"
                                className="btn btn-primary w-full"
                                disabled={loading}
                            >
                                {loading && (
                                    <span className="loading loading-spinner"></span>
                                )}
                                Generate Invite
                            </button>
                        </div>
                    </form>
                </div>
            </div>

            <div className="card w-full max-w-4xl bg-base-100 shadow-xl border border-base-200 mb-10">
                <div className="card-body">
                    <h2 className="card-title mb-4">Active Invitations</h2>
                    <div className="overflow-x-auto">
                        <table className="table">
                            <thead>
                                <tr>
                                    <th>Email</th>
                                    <th>Status</th>
                                    <th>Created At</th>
                                    <th>Expires At</th>
                                    <th>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                {invitations.map((inv) => (
                                    <tr key={inv.id}>
                                        <td>{inv.email}</td>
                                        <td>
                                            {inv.usedAt ? (
                                                <span className="badge badge-success">
                                                    Used
                                                </span>
                                            ) : new Date(inv.expiresAt) <
                                              new Date() ? (
                                                <span className="badge badge-error">
                                                    Expired
                                                </span>
                                            ) : (
                                                <span className="badge badge-info">
                                                    Active
                                                </span>
                                            )}
                                        </td>
                                        <td>
                                            {new Date(
                                                inv.createdAt
                                            ).toLocaleDateString()}
                                        </td>
                                        <td>
                                            {new Date(
                                                inv.expiresAt
                                            ).toLocaleDateString()}
                                        </td>
                                        <td>
                                            {!inv.usedAt && (
                                                <button
                                                    className="btn btn-xs btn-outline"
                                                    onClick={() =>
                                                        copyLink(inv.token)
                                                    }
                                                >
                                                    Copy Link
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                                {invitations.length === 0 && (
                                    <tr>
                                        <td colSpan={5} className="text-center">
                                            No invitations found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default InviteUser;
