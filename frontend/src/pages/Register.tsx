import { Link } from "react-router-dom";
import { useRegister } from "../hooks/RegisterContextHooks";
import { RegisterProvider } from "../contexts/RegisterContext";

const RegisterContent = () => {
    const {
        formData,
        loading,
        validating,
        error,
        tokenValid,
        token,
        handleChange,
        handleSubmit,
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
                                        />
                                    </label>
                                </div>
                                <div>
                                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                        Email
                                    </label>
                                    <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
                                        <input
                                            type="email"
                                            name="email"
                                            className="grow"
                                            value={formData.email}
                                            onChange={handleChange}
                                            required
                                            readOnly
                                        />
                                    </label>
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
                                        />
                                    </label>
                                </div>
                            </div>

                            {error && (
                                <div className="mt-4 rounded-2xl border border-error/30 bg-base-100 p-3 text-sm text-error">
                                    {error}
                                </div>
                            )}

                            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                                <button
                                    type="submit"
                                    className="btn btn-neutral w-full sm:w-auto"
                                    disabled={loading}
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
