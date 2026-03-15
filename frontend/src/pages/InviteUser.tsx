import { InviteUserProvider } from "../contexts/InviteUserContext";
import { useInviteUser } from "../hooks/InviteUserContextHooks";

const InviteUserContent = () => {
    const { loading, invitations, handleSubmit, copyLink, expireInvitation } =
        useInviteUser();

    const isExpired = (expiresAt: string) => {
        return new Date(expiresAt) < new Date();
    };

    return (
        <div className="page-shell">
            <div className="page-container">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Invitations</div>
                        <h1 className="page-title">Invite users</h1>
                        <p className="page-copy">
                            Create invite links and track their status.
                        </p>
                    </div>
                </div>

                <div className="flex flex-col gap-8">
                    <div className="panel-card rounded-[2rem] p-6 shadow-sm">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                            <div>
                                <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    New invite
                                </div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Generate a new invite link for a user.
                                </p>
                            </div>
                            <form
                                onSubmit={handleSubmit}
                                className="w-full sm:w-auto"
                            >
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

                    <div className="panel-card rounded-[2rem] p-6 shadow-sm">
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
                                            <td className="text-sm">
                                                {inv.usedAt && inv.email
                                                    ? inv.email
                                                    : "-"}
                                            </td>
                                            <td className="text-sm">
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
                                            <td className="whitespace-nowrap text-sm">
                                                {new Date(
                                                    inv.createdAt,
                                                ).toLocaleDateString()}
                                            </td>
                                            <td className="whitespace-nowrap text-sm">
                                                {new Date(
                                                    inv.expiresAt,
                                                ).toLocaleDateString()}
                                            </td>
                                            <td className="min-w-[10rem] align-top">
                                                {!inv.usedAt && (
                                                    <div className="flex min-w-[8.5rem] flex-col gap-2 sm:min-w-0 sm:flex-row">
                                                        <button
                                                            className="btn btn-sm btn-ghost w-full justify-center whitespace-nowrap sm:w-auto"
                                                            disabled={isExpired(
                                                                inv.expiresAt,
                                                            )}
                                                            onClick={() =>
                                                                copyLink(
                                                                    inv.token,
                                                                )
                                                            }
                                                        >
                                                            Copy Link
                                                        </button>
                                                        <button
                                                            className="btn btn-sm btn-error btn-outline w-full justify-center whitespace-nowrap sm:w-auto"
                                                            disabled={isExpired(
                                                                inv.expiresAt,
                                                            )}
                                                            onClick={() =>
                                                                expireInvitation(
                                                                    inv.token,
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
