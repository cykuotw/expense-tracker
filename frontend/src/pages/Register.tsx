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

const Register = () => {
    return (
        <RegisterProvider>
            <RegisterContent />
        </RegisterProvider>
    );
};

export default Register;
