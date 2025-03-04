import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { API_URL } from "../../configs/config";

export default function LoginForm() {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [feedback, setFeedback] = useState("");
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();

    const handleLoginSubmit = async (e: React.FormEvent) => {
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
        <>
            <form
                className="flex flex-col justify-center items-center gap-3 w-2/3 md:w-1/4 md:max-w-72"
                onSubmit={handleLoginSubmit}
            >
                <div className="text-2xl">Sign In</div>
                <label className="input flex items-center gap-2 w-full">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 16 16"
                        fill="currentColor"
                        className="w-4 h-4 opacity-70"
                    >
                        <path d="M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM12.735 14c.618 0 1.093-.561.872-1.139a6.002 6.002 0 0 0-11.215 0c-.22.578.254 1.139.872 1.139h9.47Z"></path>
                    </svg>
                    <input
                        type="email"
                        id="email"
                        name="email"
                        className="grow"
                        placeholder="example@your.email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                    />
                </label>
                <label className="input flex items-center gap-2 w-full">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 16 16"
                        fill="currentColor"
                        className="w-4 h-4 opacity-70"
                    >
                        <path
                            fillRule="evenodd"
                            d="M14 6a4 4 0 0 1-4.899 3.899l-1.955 1.955a.5.5 0 0 1-.353.146H5v1.5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1-.5-.5v-2.293a.5.5 0 0 1 .146-.353l3.955-3.955A4 4 0 1 1 14 6Zm-4-2a.75.75 0 0 0 0 1.5.5.5 0 0 1 .5.5.75.75 0 0 0 1.5 0 2 2 0 0 0-2-2Z"
                            clipRule="evenodd"
                        ></path>
                    </svg>
                    <input
                        type="password"
                        id="password"
                        name="password"
                        className="grow"
                        placeholder="Password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                    />
                </label>
                <button
                    type="submit"
                    className="btn btn-active btn-neutral btn-wide text-lg font-light"
                >
                    Login
                </button>
                <div id="indicator" className={`${loading ? "" : "hidden"}`}>
                    <div className="flex justify-center items-center w-full">
                        <span className="loading loading-spinner loading-md"></span>
                    </div>
                </div>
                <div
                    id="feedback"
                    className={`text-red-500 ${feedback ? "" : "hidden"}`}
                >
                    {feedback}
                </div>
            </form>
        </>
    );
}
