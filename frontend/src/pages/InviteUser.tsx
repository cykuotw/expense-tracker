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
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-28 md:pb-0">
            <div className="mx-auto w-full max-w-6xl px-4 py-10 md:py-14">
                <div className="flex flex-col gap-8">
                    <div className="space-y-3">
                        <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                            Invitations
                        </div>
                        <h1 className="text-3xl font-semibold md:text-4xl">
                            Invite users
                        </h1>
                        <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                            Create invite links and track their status.
                        </p>
                    </div>

                    <div className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                            <div>
                                <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    New invite
                                </div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Generate a new invite link for a user.
                                </p>
                            </div>
                            <form onSubmit={handleSubmit} className="w-full sm:w-auto">
                                <button
                                    type="submit"
                                    className="btn btn-neutral w-full sm:w-auto"
                                    disabled={loading}
                                >
                                    {loading && (
                                        <span className="loading loading-spinner"></span>
                                    )}
                                    Generate Invite
                                </button>
                            </form>
                        </div>
                    </div>

                    <div className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm">
                        <div className="flex items-center justify-between">
                            <h2 className="text-lg font-semibold">
                                Active invitations
                            </h2>
                            <span className="text-sm text-base-content/60">
                                {invitations.length} total
                            </span>
                        </div>
                        <div className="mt-4 overflow-x-auto">
                            <table className="table">
                                <thead>
                                    <tr>
                                        <th>Email</th>
                                        <th>Status</th>
                                        <th>Created</th>
                                        <th>Expires</th>
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
                                                    <div className="flex flex-col gap-2 sm:flex-row">
                                                        <button
                                                            className="btn btn-xs btn-ghost"
                                                            disabled={isExpired(
                                                                inv.expiresAt
                                                            )}
                                                            onClick={() =>
                                                                copyLink(
                                                                    inv.token
                                                                )
                                                            }
                                                        >
                                                            Copy Link
                                                        </button>
                                                        <button
                                                            className="btn btn-xs btn-error btn-outline"
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
                                            <td
                                                colSpan={5}
                                                className="text-center text-sm text-base-content/70"
                                            >
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
