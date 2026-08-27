import { InviteUserProvider } from "../contexts/InviteUserContext";
import { useInviteUser } from "../hooks/InviteUserContextHooks";

const InviteUserContent = () => {
    const {
        email,
        setEmail,
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
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                            <div>
                                <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    New invite
                                </div>
                                <p className="mt-2 text-sm text-base-content/70">
                                    Generate a new invite link for a user.
                                </p>
                            </div>
                            <form onSubmit={handleSubmit} className="w-full lg:max-w-xl">
                                <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                                    <div className="w-full">
                                        <label
                                            htmlFor="invitation-email"
                                            className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60"
                                        >
                                            Email (optional)
                                        </label>
                                        <input
                                            id="invitation-email"
                                            type="email"
                                            name="email"
                                            className="input input-bordered mt-2 w-full bg-base-100"
                                            value={email}
                                            onChange={(event) =>
                                                setEmail(event.target.value)
                                            }
                                            placeholder="person@example.com"
                                            autoComplete="email"
                                            aria-describedby="invitation-email-help"
                                        />
                                        <p
                                            id="invitation-email-help"
                                            className="mt-2 text-xs text-base-content/60"
                                        >
                                            Leave blank to let the recipient
                                            choose their email.
                                        </p>
                                    </div>
                                    <button
                                        type="submit"
                                        className="btn btn-neutral min-h-11 w-full sm:w-auto"
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
                                                {inv.email || "Any email"}
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
