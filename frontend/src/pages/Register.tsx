import { Link } from "react-router-dom";
import { useRegister } from "../hooks/RegisterContextHooks";
import { RegisterProvider } from "../contexts/RegisterContext";
import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import { GOOGLE_OAUTH_ENABLED } from "../configs/config";

const RegisterContent = () => {
    const {
        formData,
        loading,
        googleLoading,
        validating,
        error,
        tokenValid,
        emailBound,
        token,
        handleChange,
        handleSubmit,
        handleGoogleCredentialResponse,
    } = useRegister();

    return (
        <div className="page-shell">
            <div className="page-container max-w-4xl">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Join the group</div>
                        <h1 className="page-title">Create your account</h1>
                        <p className="page-copy">
                            Use your invite to join the group and start tracking
                            expenses.
                        </p>
                    </div>
                </div>

                {validating ? (
                    <div className="flex justify-center items-center py-12">
                        <span className="loading loading-spinner loading-lg"></span>
                    </div>
                ) : !token || !tokenValid ? (
                    <div className="panel-card rounded-[2rem] border-error/30 p-6 text-sm text-error">
                        {error ||
                            "Registration requires a valid invitation link."}
                    </div>
                ) : (
                    <div className="panel-card rounded-[2rem] p-6 md:p-8">
                        {GOOGLE_OAUTH_ENABLED ? (
                            <div className="mb-7 space-y-4">
                                <div>
                                    <h2 className="text-lg font-semibold text-base-content">
                                        Register with Google
                                    </h2>
                                    <p className="mt-1 text-sm leading-6 text-base-content/65">
                                        Choose the invited Google account. You’ll
                                        return to login after registration.
                                    </p>
                                </div>
                                <div
                                    aria-busy={googleLoading}
                                    aria-disabled={loading || googleLoading}
                                    className={
                                        loading || googleLoading
                                            ? "pointer-events-none opacity-60"
                                            : undefined
                                    }
                                >
                                    <GoogleSignInButton
                                        onCredentialResponse={
                                            handleGoogleCredentialResponse
                                        }
                                    />
                                </div>
                                <div className="flex items-center gap-3 text-xs uppercase tracking-[0.2em] text-base-content/50">
                                    <span className="h-px flex-1 bg-base-300" />
                                    or register with password
                                    <span className="h-px flex-1 bg-base-300" />
                                </div>
                            </div>
                        ) : null}
                        <form onSubmit={handleSubmit}>
                            <div className="grid gap-5 md:grid-cols-2">
                                <div>
                                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                        First name
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            type="text"
                                            name="firstname"
                                            className="grow"
                                            value={formData.firstname}
                                            onChange={handleChange}
                                            required
                                            disabled={loading || googleLoading}
                                        />
                                    </label>
                                </div>
                                <div>
                                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                        Last name
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            type="text"
                                            name="lastname"
                                            className="grow"
                                            value={formData.lastname}
                                            onChange={handleChange}
                                            required
                                            disabled={loading || googleLoading}
                                        />
                                    </label>
                                </div>
                                <div>
                                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                        Nickname (optional)
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            type="text"
                                            name="nickname"
                                            className="grow"
                                            value={formData.nickname}
                                            onChange={handleChange}
                                            disabled={loading || googleLoading}
                                        />
                                    </label>
                                </div>
                                <div>
                                    <label
                                        htmlFor="registration-email"
                                        className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60"
                                    >
                                        Email
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            id="registration-email"
                                            type="email"
                                            name="email"
                                            className="grow"
                                            value={formData.email}
                                            onChange={handleChange}
                                            required
                                            readOnly={emailBound}
                                            disabled={loading || googleLoading}
                                            autoComplete="email"
                                            aria-describedby="registration-email-help"
                                        />
                                    </label>
                                    <p
                                        id="registration-email-help"
                                        className="mt-2 text-xs text-base-content/60"
                                    >
                                        {emailBound
                                            ? "This invitation is linked to this email address."
                                            : "Enter the email address for your new account."}
                                    </p>
                                </div>
                                <div className="md:col-span-2">
                                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                        Password
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            type="password"
                                            name="password"
                                            className="grow"
                                            value={formData.password}
                                            onChange={handleChange}
                                            required
                                            minLength={8}
                                            disabled={loading || googleLoading}
                                        />
                                    </label>
                                </div>
                            </div>

                            {error && (
                                <div
                                    role="alert"
                                    className="mt-4 rounded-2xl border border-error/30 bg-base-100 p-3 text-sm text-error"
                                >
                                    {error}
                                </div>
                            )}

                            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                                <button
                                    type="submit"
                                    className="btn btn-neutral w-full sm:w-auto"
                                    disabled={loading || googleLoading}
                                >
                                    {loading && (
                                        <span className="loading loading-spinner"></span>
                                    )}
                                    Register
                                </button>
                                <Link
                                    to="/login"
                                    className="btn btn-ghost w-full sm:w-auto"
                                >
                                    Already have an account? Login
                                </Link>
                            </div>
                        </form>
                    </div>
                )}
            </div>
        </div>
    );
};

const Register = () => {
    return (
        <RegisterProvider>
            <RegisterContent />
        </RegisterProvider>
    );
};

export default Register;
