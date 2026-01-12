import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import LoginForm from "../components/auth/LoginForm";
import { LoginProvider } from "../contexts/LoginContext";

export default function Login() {
    return (
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-20 md:pb-0">
            <div className="mx-auto flex w-full max-w-4xl flex-col px-4 py-12 md:py-16">
                <div className="mb-8 space-y-3">
                    <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                        Welcome back
                    </div>
                    <h1 className="text-3xl font-semibold md:text-4xl">
                        Sign in to continue
                    </h1>
                    <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                        Access your groups, track expenses, and settle up with
                        ease.
                    </p>
                </div>

                <div className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm">
                    <LoginProvider>
                        <div className="space-y-6">
                            <LoginForm />
                            <div className="flex items-center gap-3 text-xs uppercase tracking-[0.2em] text-base-content/50">
                                <span className="h-px flex-1 bg-base-300"></span>
                                or
                                <span className="h-px flex-1 bg-base-300"></span>
                            </div>
                            <GoogleSignInButton />
                        </div>
                    </LoginProvider>
                </div>
            </div>
        </div>
    );
}
