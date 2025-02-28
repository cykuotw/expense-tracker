import { useEffect, useState } from "react";
import { API_URL } from "../../configs/config";
import useDebounce from "../../hooks/useDebounce";

export default function RegisterForm() {
    const [email, setEmail] = useState("");
    const [emailFeedback, setEmailFeedback] = useState("");
    const [loading, setLoading] = useState(false);

    const debouncedEmail = useDebounce(email, 300);

    const isValidEmail = (email: string) => {
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    };

    useEffect(() => {
        if (!debouncedEmail) {
            setEmailFeedback("");
            return;
        }

        if (!isValidEmail(debouncedEmail)) {
            setEmailFeedback("Please enter valid email");
            return;
        }

        const checkEmailValid = async () => {
            if (!email) {
                return;
            }

            setLoading(true);
            try {
                const response = await fetch(`${API_URL}/checkEmail`, {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({ email: email }),
                });
                const data = await response.json();

                if (data.exist) {
                    setEmailFeedback("Email already exists");
                } else {
                    setEmailFeedback("");
                }
            } catch (error) {
                setEmailFeedback(`Error checking email: ${error}`);
            } finally {
                setLoading(false);
            }
        };

        checkEmailValid();
    }, [debouncedEmail]);

    return (
        <div className="h-screen flex justify-center items-center">
            <form className="flex flex-col justify-center items-center gap-3 py-5 min-w-1/5">
                <div className="text-2xl">Welcome</div>
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
                        required
                        type="text"
                        name="firstname"
                        className="grow"
                        placeholder="First Name"
                    />
                </label>
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
                        required
                        type="text"
                        name="lastname"
                        className="grow"
                        placeholder="Last Name"
                    />
                </label>
                <label className="input flex items-center gap-2 w-full">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="16"
                        height="16"
                        fill="currentColor"
                        className="bi bi-suit-heart-fill"
                        viewBox="0 0 16 16"
                    >
                        <path d="M4 1c2.21 0 4 1.755 4 3.92C8 2.755 9.79 1 12 1s4 1.755 4 3.92c0 3.263-3.234 4.414-7.608 9.608a.513.513 0 0 1-.784 0C3.234 9.334 0 8.183 0 4.92 0 2.755 1.79 1 4 1"></path>
                    </svg>
                    <input
                        type="text"
                        name="nickname"
                        className="grow"
                        placeholder="Nickname (Optional)"
                    />
                </label>
                <label className="input  flex items-center gap-2 w-full">
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 16 16"
                        fill="currentColor"
                        className="w-4 h-4 opacity-70"
                    >
                        <path d="M2.5 3A1.5 1.5 0 0 0 1 4.5v.793c.026.009.051.02.076.032L7.674 8.51c.206.1.446.1.652 0l6.598-3.185A.755.755 0 0 1 15 5.293V4.5A1.5 1.5 0 0 0 13.5 3h-11Z"></path>
                        <path d="M15 6.954 8.978 9.86a2.25 2.25 0 0 1-1.956 0L1 6.954V11.5A1.5 1.5 0 0 0 2.5 13h11a1.5 1.5 0 0 0 1.5-1.5V6.954Z"></path>
                    </svg>
                    <input
                        required
                        type="text"
                        name="email"
                        value={email}
                        className="grow"
                        placeholder="example@youremail.com"
                        onChange={(e) => {
                            setEmail(e.target.value);
                        }}
                    />
                </label>
                <div
                    id="email-msg"
                    className={`-mt-2 text-xs w-full text-center text-red-500 ${
                        emailFeedback ? "" : "hidden"
                    }`}
                >
                    {emailFeedback}
                </div>
                <label className="input validator flex items-center gap-2 w-full">
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
                        required
                        type="password"
                        name="password"
                        className="grow"
                        placeholder="Password"
                        minLength={8}
                        pattern="(?=.*\d)(?=.*[a-z])(?=.*[A-Z]).{8,}"
                    />
                </label>
                <div
                    id="password-msg"
                    className="validator-hint -mt-2 text-xs w-full text-center text-red-500 hidden"
                >
                    Must be more than 8 characters, including
                    <br />
                    At least one number
                    <br />
                    At least one lowercase letter
                    <br />
                    At least one uppercase letter
                </div>
                <button
                    type="submit"
                    className="btn btn-active btn-neutral btn-wide text-lg font-light"
                >
                    Submit
                </button>
                <div id="indicator" className={`${loading ? "" : "hidden"}`}>
                    <div className="flex justify-center items-center w-full">
                        <span className="loading loading-spinner loading-md"></span>
                    </div>
                </div>
                <div id="feedback"></div>
            </form>
        </div>
    );
}
