import { useState, FormEvent } from "react";
import { API_URL } from "../configs/config";

const InviteUser = () => {
    const [email, setEmail] = useState("");
    const [token, setToken] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

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
        <div className="flex justify-center mt-10">
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
        </div>
    );
};

export default InviteUser;
