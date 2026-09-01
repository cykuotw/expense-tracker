import { FormEvent, type ReactNode, useCallback, useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import Icon from "@mdi/react";
import { mdiChevronDown } from "@mdi/js";

import ConfirmationDialog from "../components/ConfirmationDialog";
import MobilePageHeader from "../components/MobilePageHeader";
import { useAuth } from "../hooks/AuthContextHooks";
import { apiFetch, asArray, getResponseError, getResponseErrorMessage } from "../lib/api";
import {
    AdminInvitation,
    AdminManagementData,
    AdminUser,
} from "../types/admin";
import { USER_ROLES, UserRole } from "../types/role";

const EMPTY_DATA: AdminManagementData = { users: [], invitations: [] };

interface ConfirmationRequest {
    title: string;
    description: string;
    detail?: string;
    confirmLabel: string;
    tone?: "default" | "danger";
    action: () => Promise<void>;
}

function formatDate(value: string) {
    return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
        new Date(value),
    );
}

function userName(user: AdminUser) {
    const fullName = `${user.firstname} ${user.lastname}`.trim();
    return user.nickname || fullName || user.email;
}

function statusBadge(status: string) {
    const styles: Record<string, string> = {
        active: "ui-badge-success",
        inactive: "ui-badge-destructive",
        invited: "ui-badge-info",
        expired: "ui-badge-warning",
        used: "ui-badge-neutral",
    };
    return `ui-badge ${styles[status] ?? "ui-badge-ghost"}`;
}

const PROTECTED_ADMIN_MESSAGE =
    "The system owner is protected for account recovery and cannot be managed here.";

async function getAdminMutationError(response: Response, fallback: string) {
    const error = await getResponseError(response, fallback);
    return error.code === "PROTECTED_ADMIN"
        ? PROTECTED_ADMIN_MESSAGE
        : error.message;
}

interface UserActionsProps {
    user: AdminUser;
    currentUserID: string | null;
    busy: boolean;
    onStatusChange: (user: AdminUser) => Promise<void>;
    onRoleChange: (user: AdminUser, role: UserRole) => Promise<void>;
}

function UserActions({
    user,
    currentUserID,
    busy,
    onStatusChange,
    onRoleChange,
}: UserActionsProps) {
    const isSelf = user.id === currentUserID;

    if (user.isProtectedAdmin) {
        return (
            <div
                className="rounded-2xl border border-info/30 bg-info/10 p-4"
                aria-label={`System owner details for ${user.email}`}
            >
                <dl className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-foreground/55">
                            Role
                        </dt>
                        <dd className="mt-1 font-medium">Administrator</dd>
                    </div>
                    <div>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-foreground/55">
                            Status
                        </dt>
                        <dd className="mt-1 font-medium">Active</dd>
                    </div>
                </dl>
                <p className="mt-3 text-sm text-foreground/70">
                    {PROTECTED_ADMIN_MESSAGE}
                </p>
            </div>
        );
    }

    return (
        <div className="flex min-w-[11rem] flex-col gap-2">
            <label className="sr-only" htmlFor={`role-${user.id}`}>
                Role for {user.email}
            </label>
            <select
                id={`role-${user.id}`}
                className="ui-select min-h-11 w-full bg-background"
                value={user.role}
                disabled={busy || isSelf}
                onChange={(event) =>
                    void onRoleChange(user, event.target.value as UserRole)
                }
                title={isSelf ? "You cannot change your own role" : undefined}
            >
                <option value={USER_ROLES.user}>Regular user</option>
                <option value={USER_ROLES.admin}>Administrator</option>
            </select>
            <button
                type="button"
                className={`ui-button min-h-11 w-full ${
                    user.isActive ? "ui-button-destructive ui-button-outline" : "ui-button-success"
                }`}
                disabled={busy || isSelf}
                onClick={() => void onStatusChange(user)}
                title={
                    isSelf ? "You cannot deactivate your own account" : undefined
                }
            >
                {busy ? (
                    <span className="ui-spinner ui-spinner-sm" />
                ) : user.isActive ? (
                    "Disable account"
                ) : (
                    "Activate account"
                )}
            </button>
        </div>
    );
}

interface UserCardProps extends Omit<UserActionsProps, "busy"> {
    busy: boolean;
}

function UserCard(props: UserCardProps) {
    const { user } = props;
    return (
        <article
            className={`rounded-3xl border bg-background p-5 ${
                user.isProtectedAdmin
                    ? "border-info/40 ring-1 ring-info/15"
                    : "border-border"
            }`}
        >
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                    <h3 className="truncate font-semibold text-foreground">
                        {userName(user)}
                    </h3>
                    <p className="break-all text-sm text-foreground/65">
                        {user.email}
                    </p>
                </div>
                <div className="flex flex-wrap gap-2" aria-label="Account labels">
                    <span
                        className={statusBadge(
                            user.isActive ? "active" : "inactive",
                        )}
                    >
                        {user.isActive ? "Active" : "Disabled"}
                    </span>
                    {user.role === USER_ROLES.admin ? (
                        <span className="ui-badge ui-badge-outline">Administrator</span>
                    ) : (
                        <span className="ui-badge ui-badge-outline">Regular user</span>
                    )}
                    {user.isProtectedAdmin ? (
                        <span className="ui-badge ui-badge-info ui-badge-outline">
                            System owner
                        </span>
                    ) : null}
                </div>
            </div>
            <p className="mt-3 text-xs text-foreground/55">
                Joined {formatDate(user.createTime)}
            </p>
            <div className="mt-5">
                <UserActions {...props} />
            </div>
        </article>
    );
}

interface CollapsibleSectionProps {
    id: string;
    title: string;
    description: string;
    count: number;
    expanded: boolean;
    onToggle: () => void;
    children: ReactNode;
}

function CollapsibleSection({
    id,
    title,
    description,
    count,
    expanded,
    onToggle,
    children,
}: CollapsibleSectionProps) {
    const contentID = `${id}-content`;

    return (
        <section
            className="panel-card rounded-[2rem] p-5 md:p-6"
            aria-labelledby={id}
        >
            <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                    <h2 id={id} className="text-lg font-semibold">
                        {title}
                    </h2>
                    <p className="mt-1 text-sm text-foreground/65">
                        {description}
                    </p>
                </div>
                <button
                    type="button"
                    className="ui-button ui-button-outline min-h-11 shrink-0 gap-2 px-3 transition-colors hover:bg-base-200"
                    aria-controls={contentID}
                    aria-expanded={expanded}
                    aria-label={`${expanded ? "Hide" : "Show"} ${title} section`}
                    onClick={onToggle}
                >
                    <span className="ui-badge ui-badge-ghost" aria-hidden="true">
                        {count}
                    </span>
                    <span aria-hidden="true">{expanded ? "Hide" : "Show"}</span>
                    <Icon
                        path={mdiChevronDown}
                        size={0.85}
                        aria-hidden="true"
                        className={`transition-transform ${expanded ? "rotate-180" : ""}`}
                    />
                </button>
            </div>
            {expanded ? <div id={contentID}>{children}</div> : null}
        </section>
    );
}

interface UserSectionProps {
    id: string;
    title: string;
    description: string;
    users: AdminUser[];
    expanded: boolean;
    onToggle: () => void;
    currentUserID: string | null;
    busyID: string | null;
    onStatusChange: (user: AdminUser) => Promise<void>;
    onRoleChange: (user: AdminUser, role: UserRole) => Promise<void>;
}

function UserSection({
    id,
    title,
    description,
    users,
    expanded,
    onToggle,
    currentUserID,
    busyID,
    onStatusChange,
    onRoleChange,
}: UserSectionProps) {
    return (
        <CollapsibleSection
            id={id}
            title={title}
            description={description}
            count={users.length}
            expanded={expanded}
            onToggle={onToggle}
        >
            {users.length === 0 ? (
                <p className="mt-5 text-sm text-foreground/65">
                    No {title.toLowerCase()} found.
                </p>
            ) : (
                <div className="mt-5 grid gap-4 lg:grid-cols-2">
                    {users.map((user) => (
                        <UserCard
                            key={user.id}
                            user={user}
                            currentUserID={currentUserID}
                            busy={busyID === user.id}
                            onStatusChange={onStatusChange}
                            onRoleChange={onRoleChange}
                        />
                    ))}
                </div>
            )}
        </CollapsibleSection>
    );
}

interface InvitationActionsProps {
    invitation: AdminInvitation;
    busy: boolean;
    onCopy: (invitation: AdminInvitation) => Promise<void>;
    onExpire: (invitation: AdminInvitation) => Promise<void>;
}

function InvitationActions({
    invitation,
    busy,
    onCopy,
    onExpire,
}: InvitationActionsProps) {
    if (invitation.status !== "invited") {
        return <span className="text-sm text-foreground/55">No actions</span>;
    }
    return (
        <div className="flex min-w-[9rem] flex-col gap-2 sm:flex-row">
            <button
                type="button"
                className="ui-button ui-button-ghost min-h-11"
                disabled={busy}
                onClick={() => void onCopy(invitation)}
            >
                {busy ? (
                    <span className="ui-spinner ui-spinner-sm" />
                ) : null}
                Copy link
            </button>
            <button
                type="button"
                className="ui-button ui-button-destructive ui-button-outline min-h-11"
                disabled={busy}
                onClick={() => void onExpire(invitation)}
            >
                {busy ? (
                    <span className="ui-spinner ui-spinner-sm" />
                ) : null}
                Expire
            </button>
        </div>
    );
}

export default function AdminUsers() {
    const { userID } = useAuth();
    const [data, setData] = useState<AdminManagementData>(EMPTY_DATA);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [busyID, setBusyID] = useState<string | null>(null);
    const [inviteEmail, setInviteEmail] = useState("");
    const [creatingInvite, setCreatingInvite] = useState(false);
    const [confirmation, setConfirmation] =
        useState<ConfirmationRequest | null>(null);
    const [expandedSections, setExpandedSections] = useState({
        administrators: true,
        regularUsers: false,
        invitations: false,
    });

    const administrators: AdminUser[] = [];
    const regularUsers: AdminUser[] = [];
    for (const user of data.users) {
        if (user.role === USER_ROLES.admin) {
            administrators.push(user);
        } else {
            regularUsers.push(user);
        }
    }

    const load = useCallback(async (showLoading = true) => {
        if (showLoading) setLoading(true);
        setError("");
        try {
            const response = await apiFetch("/admin/users");
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to load user management data",
                    ),
                );
            }
            const responseData = (await response.json()) as AdminManagementData;
            setData({
                users: asArray(responseData.users),
                invitations: asArray(responseData.invitations),
            });
        } catch (loadError) {
            setError(
                loadError instanceof Error
                    ? loadError.message
                    : "Unable to load user management data",
            );
        } finally {
            if (showLoading) setLoading(false);
        }
    }, []);

    useEffect(() => {
        void load();
    }, [load]);

    const performStatusUpdate = async (user: AdminUser) => {
        if (user.isProtectedAdmin) {
            toast.error(PROTECTED_ADMIN_MESSAGE);
            return;
        }
        const nextActive = !user.isActive;
        setBusyID(user.id);
        try {
            const response = await apiFetch(`/admin/users/${user.id}/status`, {
                method: "PATCH",
                body: JSON.stringify({ isActive: nextActive }),
            });
            if (!response.ok) {
                throw new Error(
                    await getAdminMutationError(
                        response,
                        "Unable to update account status",
                    ),
                );
            }
            setData((current) => ({
                ...current,
                users: current.users.map((entry) =>
                    entry.id === user.id
                        ? { ...entry, isActive: nextActive }
                        : entry,
                ),
            }));
            toast.success(nextActive ? "Account activated" : "Account disabled");
        } catch (updateError) {
            toast.error(
                updateError instanceof Error
                    ? updateError.message
                    : "Unable to update account status",
            );
        } finally {
            setBusyID(null);
        }
    };

    const requestStatusUpdate = async (user: AdminUser) => {
        if (user.isProtectedAdmin) {
            toast.error(PROTECTED_ADMIN_MESSAGE);
            return;
        }
        const nextActive = !user.isActive;
        setConfirmation({
            title: nextActive ? "Activate this account?" : "Disable this account?",
            description: nextActive
                ? "This user will be able to sign in and refresh their session again."
                : "This user will be signed out when their session needs to refresh and will not be able to sign in.",
            detail: user.email,
            confirmLabel: nextActive ? "Activate account" : "Disable account",
            tone: nextActive ? "default" : "danger",
            action: () => performStatusUpdate(user),
        });
    };

    const performRoleUpdate = async (user: AdminUser, role: UserRole) => {
        if (user.isProtectedAdmin) {
            toast.error(PROTECTED_ADMIN_MESSAGE);
            return;
        }
        if (role === user.role) return;
        setBusyID(user.id);
        try {
            const response = await apiFetch(`/admin/users/${user.id}/role`, {
                method: "PATCH",
                body: JSON.stringify({ role }),
            });
            if (!response.ok) {
                throw new Error(
                    await getAdminMutationError(
                        response,
                        "Unable to update user role",
                    ),
                );
            }
            setData((current) => ({
                ...current,
                users: current.users.map((entry) =>
                    entry.id === user.id ? { ...entry, role } : entry,
                ),
            }));
            toast.success("User role updated");
        } catch (updateError) {
            toast.error(
                updateError instanceof Error
                    ? updateError.message
                    : "Unable to update user role",
            );
        } finally {
            setBusyID(null);
        }
    };

    const requestRoleUpdate = async (user: AdminUser, role: UserRole) => {
        if (user.isProtectedAdmin) {
            toast.error(PROTECTED_ADMIN_MESSAGE);
            return;
        }
        if (role === user.role) return;
        const currentLabel =
            user.role === USER_ROLES.admin ? "Administrator" : "Regular user";
        const nextLabel =
            role === USER_ROLES.admin ? "Administrator" : "Regular user";
        setConfirmation({
            title: "Change this user’s role?",
            description:
                role === USER_ROLES.admin
                    ? "Administrators can manage users, roles, account access, and invitations."
                    : "This user will lose access to administrative features immediately.",
            detail: `${user.email} · ${currentLabel} → ${nextLabel}`,
            confirmLabel: `Change to ${nextLabel}`,
            tone: role === USER_ROLES.admin ? "default" : "danger",
            action: () => performRoleUpdate(user, role),
        });
    };

    const createInvitation = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setCreatingInvite(true);
        try {
            const response = await apiFetch("/invitations", {
                method: "POST",
                body: JSON.stringify({ email: inviteEmail.trim() }),
            });
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to create invitation",
                    ),
                );
            }
            setInviteEmail("");
            toast.success("Invitation created");
            await load(false);
        } catch (createError) {
            toast.error(
                createError instanceof Error
                    ? createError.message
                    : "Unable to create invitation",
            );
        } finally {
            setCreatingInvite(false);
        }
    };

    const copyInvitation = async (invitation: AdminInvitation) => {
        const clipboard = navigator.clipboard;
        if (!clipboard?.writeText) {
            toast.error(
                "Copying isn't available in this browser. Please use a supported browser and try again.",
            );
            return;
        }

        setBusyID(invitation.id);
        try {
            const response = await apiFetch(
                `/admin/invitations/${invitation.id}/link`,
                { method: "POST" },
            );
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to retrieve invitation link",
                    ),
                );
            }
            const value = (await response.json()) as { token: string };
            try {
                await clipboard.writeText(
                    `${window.location.origin}/register#${value.token}`,
                );
                toast.success("New invitation link copied; any older link no longer works.");
            } catch {
                toast.error(
                    "We couldn't copy the invitation link. Check your browser's clipboard permission and try again.",
                );
            }
        } catch (copyError) {
            toast.error(
                copyError instanceof Error
                    ? copyError.message
                    : "Unable to copy invitation link",
            );
        } finally {
            setBusyID(null);
        }
    };

    const performInvitationExpiry = async (invitation: AdminInvitation) => {
        setBusyID(invitation.id);
        try {
            const response = await apiFetch(
                `/admin/invitations/${invitation.id}/expire`,
                { method: "POST" },
            );
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to expire invitation",
                    ),
                );
            }
            setData((current) => ({
                ...current,
                invitations: current.invitations.map((entry) =>
                    entry.id === invitation.id
                        ? { ...entry, status: "expired" }
                        : entry,
                ),
            }));
            toast.success("Invitation expired");
        } catch (expireError) {
            toast.error(
                expireError instanceof Error
                    ? expireError.message
                    : "Unable to expire invitation",
            );
        } finally {
            setBusyID(null);
        }
    };

    const requestInvitationExpiry = async (invitation: AdminInvitation) => {
        setConfirmation({
            title: "Expire this invitation?",
            description:
                "The existing registration link will stop working immediately. This action cannot be undone.",
            detail: invitation.email || "Invitation for any email",
            confirmLabel: "Expire invitation",
            tone: "danger",
            action: () => performInvitationExpiry(invitation),
        });
    };

    const confirmAction = async () => {
        if (!confirmation) return;
        await confirmation.action();
        setConfirmation(null);
    };

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container">
                <MobilePageHeader
                    title="User management"
                    backTo="/account"
                    backLabel="Back to account"
                />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Administration</div>
                        <h1 className="page-title">User management</h1>
                        <p className="page-copy">
                            Manage account access, roles, and invitations in one place.
                        </p>
                    </div>
                </div>

                {loading ? (
                    <div className="panel-card flex min-h-48 items-center justify-center rounded-[2rem]" aria-label="Loading users">
                        <span className="ui-spinner ui-spinner-lg" />
                    </div>
                ) : error ? (
                    <div className="panel-card rounded-[2rem] border-destructive/30 p-6" role="alert">
                        <p className="text-destructive">{error}</p>
                        <button type="button" className="ui-button ui-button-primary mt-4 min-h-11" onClick={() => void load()}>
                            Try again
                        </button>
                    </div>
                ) : (
                    <div className="space-y-5 md:space-y-8">
                        <UserSection
                            id="administrators-heading"
                            title="Administrators"
                            description="Accounts that can manage users, roles, access, and invitations."
                            users={administrators}
                            expanded={expandedSections.administrators}
                            onToggle={() =>
                                setExpandedSections((current) => ({
                                    ...current,
                                    administrators: !current.administrators,
                                }))
                            }
                            currentUserID={userID}
                            busyID={busyID}
                            onStatusChange={requestStatusUpdate}
                            onRoleChange={requestRoleUpdate}
                        />

                        <UserSection
                            id="regular-users-heading"
                            title="Regular users"
                            description="Accounts with standard expense-tracking access."
                            users={regularUsers}
                            expanded={expandedSections.regularUsers}
                            onToggle={() =>
                                setExpandedSections((current) => ({
                                    ...current,
                                    regularUsers: !current.regularUsers,
                                }))
                            }
                            currentUserID={userID}
                            busyID={busyID}
                            onStatusChange={requestStatusUpdate}
                            onRoleChange={requestRoleUpdate}
                        />

                        <CollapsibleSection
                            id="invitations-heading"
                            title="Invitations"
                            description="Create an email-bound invite or leave the email blank for a generic link."
                            count={data.invitations.length}
                            expanded={expandedSections.invitations}
                            onToggle={() =>
                                setExpandedSections((current) => ({
                                    ...current,
                                    invitations: !current.invitations,
                                }))
                            }
                        >
                            <div className="mt-5 flex flex-col gap-5 border-b border-border pb-6 lg:flex-row lg:items-end lg:justify-between">
                                <div>
                                    <p className="text-sm text-foreground/65">
                                        Create a new invitation.
                                    </p>
                                </div>
                                <form onSubmit={createInvitation} className="w-full lg:max-w-xl">
                                    <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                                        <div className="w-full">
                                            <label
                                                htmlFor="invitation-email"
                                                className="text-xs font-semibold uppercase tracking-[0.18em] text-foreground/60"
                                            >
                                                Email (optional)
                                            </label>
                                            <input
                                                id="invitation-email"
                                                type="email"
                                                className="ui-input-shell mt-2 min-h-11 w-full bg-background"
                                                value={inviteEmail}
                                                onChange={(event) => setInviteEmail(event.target.value)}
                                                placeholder="person@example.com"
                                                autoComplete="email"
                                                aria-describedby="invitation-email-help"
                                            />
                                            <span id="invitation-email-help" className="sr-only">
                                                Leave blank to create an invitation that accepts any email.
                                            </span>
                                        </div>
                                        <button
                                            type="submit"
                                            className="ui-button ui-button-primary min-h-11 w-full whitespace-nowrap sm:w-auto"
                                            disabled={creatingInvite}
                                        >
                                            {creatingInvite ? (
                                                <span className="ui-spinner ui-spinner-sm" />
                                            ) : null}
                                            {creatingInvite ? "Creating…" : "Create invite"}
                                        </button>
                                    </div>
                                </form>
                            </div>
                            {data.invitations.length === 0 ? (
                                <p className="mt-5 text-sm text-foreground/65">No invitations found.</p>
                            ) : (
                                <div className="mt-5 divide-y divide-base-300">
                                    {data.invitations.map((invitation) => (
                                        <article key={invitation.id} className="flex flex-col gap-4 py-5 first:pt-0 last:pb-0 md:flex-row md:items-center md:justify-between">
                                            <div>
                                                <div className="flex flex-wrap items-center gap-2">
                                                    <h3 className="font-medium text-foreground">{invitation.email || "Any email"}</h3>
                                                    <span className={statusBadge(invitation.status)}>{invitation.status}</span>
                                                </div>
                                                <p className="mt-1 text-xs text-foreground/55">
                                                    Created {formatDate(invitation.createdAt)} · Expires {formatDate(invitation.expiresAt)}
                                                </p>
                                            </div>
                                            <InvitationActions
                                                invitation={invitation}
                                                busy={busyID === invitation.id}
                                                onCopy={copyInvitation}
                                                onExpire={requestInvitationExpiry}
                                            />
                                        </article>
                                    ))}
                                </div>
                            )}
                        </CollapsibleSection>
                    </div>
                )}
            </div>
            {confirmation ? (
                <ConfirmationDialog
                    title={confirmation.title}
                    description={confirmation.description}
                    detail={confirmation.detail}
                    confirmLabel={confirmation.confirmLabel}
                    tone={confirmation.tone}
                    busy={busyID !== null}
                    onCancel={() => setConfirmation(null)}
                    onConfirm={() => void confirmAction()}
                />
            ) : null}
        </div>
    );
}
