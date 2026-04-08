import { useCallback } from "react";
import { toast } from "react-hot-toast";
import { useNavigate } from "react-router-dom";
import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import { GOOGLE_OAUTH_ENABLED } from "../configs/config";
import LoginForm from "../components/auth/LoginForm";
import { LoginProvider } from "../contexts/LoginContext";
import { useAuth } from "../hooks/AuthContextHooks";
import { apiFetch, getResponseErrorMessage } from "../lib/api";

export default function Login() {
    const navigate = useNavigate();
    const { markLoggedIn } = useAuth();

    const handleGoogleCredentialResponse = useCallback(
        async (response: GoogleCredentialResponse) => {
            if (!response.credential?.trim()) {
                toast.error("Google sign-in did not return a credential.");
                return;
            }

            try {
                const exchangeResponse = await apiFetch(
                    "/auth/google/exchange",
                    {
                        method: "POST",
                        headers: {
                            Authorization: `Bearer ${response.credential}`,
                        },
                    },
                    { authMode: "none" },
                );

                if (!exchangeResponse.ok) {
                    throw new Error(
                        await getResponseErrorMessage(
                            exchangeResponse,
                            "Google sign-in failed",
                        ),
                    );
                }

                const loggedIn = await markLoggedIn();
                if (!loggedIn) {
                    throw new Error(
                        "Google sign-in succeeded, but session refresh failed.",
                    );
                }

                navigate("/", { replace: true });
            } catch (error) {
                toast.error((error as Error).message);
            }
        },
        [markLoggedIn, navigate],
    );

    return (
        <div className="page-shell">
            <div className="page-container max-w-5xl">
                <div className="grid gap-6 lg:grid-cols-[0.95fr_1.05fr]">
                    <section className="panel-card rounded-[2rem] p-8 md:p-10">
                        <div className="page-eyebrow">Login</div>
                        <h1 className="mt-4 text-4xl font-bold tracking-[-0.05em] text-base-content md:text-5xl">
                            Sign in to continue
                        </h1>
                        <p className="mt-4 max-w-md text-sm leading-7 text-base-content/70 md:text-base">
                            Access your groups and expenses.
                        </p>
                        <div className="mt-8 grid gap-4 sm:grid-cols-2">
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="section-label">Groups</div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Open a group and review balances.
                                </p>
                            </div>
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="section-label">Expenses</div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Add expenses and update shared costs.
                                </p>
                            </div>
                        </div>
                    </section>

                    <div className="panel-card rounded-[2rem] p-6 md:p-8">
                        <LoginProvider>
                            <div className="space-y-6">
                                <LoginForm />
                                {GOOGLE_OAUTH_ENABLED ? (
                                    <>
                                        <div className="flex items-center gap-3 text-xs uppercase tracking-[0.2em] text-base-content/50">
                                            <span className="h-px flex-1 bg-base-300"></span>
                                            or
                                            <span className="h-px flex-1 bg-base-300"></span>
                                        </div>
                                        <GoogleSignInButton
                                            onCredentialResponse={
                                                handleGoogleCredentialResponse
                                            }
                                        />
                                    </>
                                ) : null}
                            </div>
                        </LoginProvider>
                    </div>
                </div>
            </div>
        </div>
    );
}
