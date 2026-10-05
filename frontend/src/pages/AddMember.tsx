import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";
import MobilePageHeader from "../components/MobilePageHeader";
import DesktopBackLink from "../components/DesktopBackLink";
import { GroupMemberManager } from "../components/group/GroupMemberManager";
import { AddMemberProvider } from "../contexts/AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";
import { usePWAUpdateBlocker } from "../hooks/usePWAUpdateBlocker";
import { useEffect, useState } from "react";
import { apiFetch, getResponseErrorMessage } from "../lib/api";
import ArchivedGroupMessage from "../components/group/ArchivedGroupMessage";

const AddMemberContent = () => {
    usePWAUpdateBlocker(true);
    const { groupId, loading } = useAddMember();
    const [archived, setArchived] = useState(false);
    const [statusReady, setStatusReady] = useState(false);
    const [statusError, setStatusError] = useState<string | null>(null);
    const [revision, setRevision] = useState(0);
    useEffect(() => {
        if (!groupId) return;
        const controller = new AbortController();
        setStatusReady(false);
        setStatusError(null);
        const loadStatus = async () => {
            try {
                const response = await apiFetch(`/group/${encodeURIComponent(groupId)}`, { signal: controller.signal });
                if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not load this group. Try again."));
                const group = await response.json();
                if (!controller.signal.aborted) {
                    setArchived(group.isActive === false);
                    setStatusReady(true);
                }
            } catch (cause) {
                if (!controller.signal.aborted) setStatusError(cause instanceof Error ? cause.message : "Could not load this group. Try again.");
            }
        };
        void loadStatus();
        return () => controller.abort();
    }, [groupId, revision]);
    if (groupId && !statusReady) return <div className="page-shell"><div className="page-container max-w-4xl">
        <DesktopBackLink to={`/group/${groupId}`} label="Back to group" />
        {statusError ? <div role="alert"><p>{statusError}</p><button type="button" className="ui-button ui-button-outline mt-4" onClick={() => setRevision((value) => value + 1)}>Try again</button></div> : <p role="status">Loading group…</p>}
    </div></div>;
    if (archived && groupId) return <ArchivedGroupMessage groupId={groupId} />;

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title="Add members"
                    backTo={groupId ? `/group/${groupId}` : "/"}
                    backLabel={groupId ? "Back to group" : "Back to groups"}
                    action={
                        loading ? (
                            <span className="ui-spinner ui-spinner-sm" role="status" aria-label="Updating members" />
                        ) : (
                            <button
                                type="submit"
                                form="member-selection-form"
                                className="ui-button ui-button-primary min-h-12 min-w-12 px-3"
                                aria-label="Update members"
                            >
                                <Icon path={mdiCheckBold} size={1} aria-hidden="true" />
                            </button>
                        )
                    }
                />
                <DesktopBackLink to={groupId ? `/group/${groupId}` : "/"} label={groupId ? "Back to group" : "Back to groups"} />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group Members</div>
                        <h1 className="page-title">Add members</h1>
                        <p className="page-copy">
                            Add existing friends or invite a new person by
                            email.
                        </p>
                    </div>
                </div>

                <GroupMemberManager />
            </div>
        </div>
    );
};

export default function AddMember() {
    return (
        <AddMemberProvider>
            <AddMemberContent />
        </AddMemberProvider>
    );
}
