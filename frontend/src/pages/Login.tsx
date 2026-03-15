import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import LoginForm from "../components/auth/LoginForm";
import { LoginProvider } from "../contexts/LoginContext";

export default function Login() {
    return (
        <div className="page-shell">
            <div className="page-container max-w-5xl">
                <div className="grid gap-6 lg:grid-cols-[0.95fr_1.05fr]">
                    <section className="panel-card rounded-[2rem] p-8 md:p-10">
                        <div className="page-eyebrow">Welcome back</div>
                        <h1 className="mt-4 text-4xl font-bold tracking-[-0.05em] text-base-content md:text-5xl">
                            Sign in and keep the numbers calm
                        </h1>
                        <p className="mt-4 max-w-md text-sm leading-7 text-base-content/70 md:text-base">
                            Access your groups, review balances, and settle up
                            without losing track of who paid what.
                        </p>
                        <div className="mt-8 grid gap-4 sm:grid-cols-2">
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="section-label">Track</div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Review group balances and recent expenses in
                                    one place.
                                </p>
                            </div>
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="section-label">Settle</div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Update shared costs and settle balances when
                                    plans wrap up.
                                </p>
                            </div>
                        </div>
                    </section>

                    <div className="panel-card rounded-[2rem] p-6 md:p-8">
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
        </div>
    );
}
