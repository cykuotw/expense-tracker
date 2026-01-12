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
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-20 md:pb-0">
            <div className="mx-auto flex w-full max-w-4xl flex-col px-4 py-12 md:py-16">
                <div className="mb-8 space-y-3">
                    <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                        Join the group
                    </div>
                    <h1 className="text-3xl font-semibold md:text-4xl">
                        Create your account
                    </h1>
                    <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                        Use your invite to join the group and start tracking
                        expenses.
                    </p>
                </div>

                {validating ? (
                    <div className="flex justify-center items-center py-12">
                        <span className="loading loading-spinner loading-lg"></span>
                    </div>
                ) : !token || !tokenValid ? (
                    <div className="rounded-3xl border border-error/30 bg-base-100/90 p-6 text-sm text-error shadow-sm">
                        {error ||
                            "Registration requires a valid invitation link."}
                    </div>
                ) : (
                    <div className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm">
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
