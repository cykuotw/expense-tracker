import { InviteUserProvider } from "../contexts/InviteUserContext";
import { useInviteUser } from "../hooks/InviteUserContextHooks";

const InviteUserContent = () => {
    const {
        loading,
        invitations,
        handleSubmit,
        copyLink,
        expireInvitation,
    } = useInviteUser();

    const isExpired = (expiresAt: string) => {
        return new Date(expiresAt) < new Date();
    };

    return (
        <div className="flex flex-col items-center mt-10 gap-10">
            <div className="card w-96 bg-base-100 shadow-xl border border-base-200">
                <div className="card-body">
                    <h2 className="card-title justify-center mb-4">
                        Invite User
                    </h2>
                    <form onSubmit={handleSubmit}>
                        <div className="card-actions justify-end mt-6">
                            <button
                                type="submit"
                                className="btn btn-primary w-full"
                                disabled={loading}
                            >
                                {loading && (
                                    <span className="loading loading-spinner"></span>
                                )}
                                Generate Invite
                            </button>
                        </div>
                    </form>
                </div>
            </div>

            <div className="card w-full max-w-4xl bg-base-100 shadow-xl border border-base-200 mb-10">
                <div className="card-body">
                    <h2 className="card-title mb-4">Active Invitations</h2>
                    <div className="overflow-x-auto">
                        <table className="table">
                            <thead>
                                <tr>
                                    <th>Email</th>
                                    <th>Status</th>
                                    <th>Created At</th>
                                    <th>Expires At</th>
                                    <th>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                {invitations.map((inv) => (
                                    <tr key={inv.id}>
                                        <td>
                                            {inv.usedAt && inv.email
                                                ? inv.email
                                                : "-"}
                                        </td>
                                        <td>
                                            {inv.usedAt ? (
                                                <span className="badge badge-success">
                                                    Used
                                                </span>
                                            ) : isExpired(inv.expiresAt) ? (
                                                <span className="badge badge-error">
                                                    Expired
                                                </span>
                                            ) : (
                                                <span className="badge badge-info">
                                                    Active
                                                </span>
                                            )}
                                        </td>
                                        <td>
                                            {new Date(
                                                inv.createdAt
                                            ).toLocaleDateString()}
                                        </td>
                                        <td>
                                            {new Date(
                                                inv.expiresAt
                                            ).toLocaleDateString()}
                                        </td>
                                        <td>
                                            {!inv.usedAt && (
                                                <div className="flex gap-2">
                                                    <button
                                                        className="btn btn-xs btn-outline"
                                                        disabled={isExpired(
                                                            inv.expiresAt
                                                        )}
                                                        onClick={() =>
                                                            copyLink(inv.token)
                                                        }
                                                    >
                                                        Copy Link
                                                    </button>
                                                    <button
                                                        className="btn btn-xs btn-warning"
                                                        disabled={isExpired(
                                                            inv.expiresAt
                                                        )}
                                                        onClick={() =>
                                                            expireInvitation(
                                                                inv.token
                                                            )
                                                        }
                                                    >
                                                        Expire
                                                    </button>
                                                </div>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                                {invitations.length === 0 && (
                                    <tr>
                                        <td colSpan={5} className="text-center">
                                            No invitations found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    );
};

const InviteUser = () => {
    return (
        <InviteUserProvider>
            <InviteUserContent />
        </InviteUserProvider>
    );
};

export default InviteUser;
