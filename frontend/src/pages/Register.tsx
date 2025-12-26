import { useState, useEffect, FormEvent } from "react";
import { useSearchParams, useNavigate, Link } from "react-router-dom";
import { API_URL } from "../configs/config";

const Register = () => {
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

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
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

    if (validating) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="loading loading-spinner loading-lg"></span>
            </div>
        );
    }

    if (!token || !tokenValid) {
        return (
            <div className="flex justify-center mt-10">
                <div className="alert alert-error w-96">
                    <span>
                        {error ||
                            "Registration requires a valid invitation link."}
                    </span>
                </div>
            </div>
        );
    }

    return (
        <div className="flex justify-center mt-10">
            <div className="card w-96 bg-base-100 shadow-xl border border-base-200">
                <div className="card-body">
                    <h2 className="card-title justify-center mb-4">Register</h2>
                    <form onSubmit={handleSubmit}>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">First Name</span>
                            </label>
                            <input
                                type="text"
                                name="firstname"
                                className="input input-bordered w-full"
                                value={formData.firstname}
                                onChange={handleChange}
                                required
                            />
                        </div>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">Last Name</span>
                            </label>
                            <input
                                type="text"
                                name="lastname"
                                className="input input-bordered w-full"
                                value={formData.lastname}
                                onChange={handleChange}
                                required
                            />
                        </div>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">
                                    Nickname (Optional)
                                </span>
                            </label>
                            <input
                                type="text"
                                name="nickname"
                                className="input input-bordered w-full"
                                value={formData.nickname}
                                onChange={handleChange}
                            />
                        </div>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">Email</span>
                            </label>
                            <input
                                type="email"
                                name="email"
                                className="input input-bordered w-full"
                                value={formData.email}
                                onChange={handleChange}
                                required
                                readOnly
                            />
                        </div>
                        <div className="form-control w-full">
                            <label className="label">
                                <span className="label-text">Password</span>
                            </label>
                            <input
                                type="password"
                                name="password"
                                className="input input-bordered w-full"
                                value={formData.password}
                                onChange={handleChange}
                                required
                                minLength={8}
                            />
                        </div>

                        {error && (
                            <div className="alert alert-error mt-4 text-sm">
                                <span>{error}</span>
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
                                Register
                            </button>
                        </div>
                        <div className="text-center mt-4">
                            <Link to="/login" className="link link-primary">
                                Already have an account? Login
                            </Link>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    );
};

export default Register;
