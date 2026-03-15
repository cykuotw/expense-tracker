import { useLogin } from "../../hooks/LoginContextHooks";

export default function LoginForm() {
    const {
        email,
        password,
        feedback,
        loading,
        setEmail,
        setPassword,
        handleLoginSubmit,
    } = useLogin();

    return (
        <form
            className="mx-auto flex w-full max-w-md flex-col gap-4"
            onSubmit={handleLoginSubmit}
        >
            <div>
                <div className="section-label">Sign in</div>
                <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-base-content">
                    Continue to your workspace
                </h2>
            </div>

            <div>
                <label
                    htmlFor="email"
                    className="mb-2 block text-sm font-semibold text-base-content/80"
                >
                    Email
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
            </div>

            <div>
                <label
                    htmlFor="password"
                    className="mb-2 block text-sm font-semibold text-base-content/80"
                >
                    Password
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
            </div>

            <button type="submit" className="btn btn-neutral w-full text-lg">
                Login
            </button>
            <div id="indicator" className={`${loading ? "" : "hidden"}`}>
                <div className="flex justify-center items-center w-full">
                    <span className="loading loading-spinner loading-md"></span>
                </div>
            </div>
            <div
                id="feedback"
                className={`rounded-2xl border border-error/20 bg-error/8 px-4 py-3 text-sm text-error ${
                    feedback ? "" : "hidden"
                }`}
            >
                {feedback}
            </div>
        </form>
    );
}
