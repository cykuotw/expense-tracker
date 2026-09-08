import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";
import MobilePageHeader from "../components/MobilePageHeader";
import { GroupMemberManager } from "../components/group/GroupMemberManager";
import { AddMemberProvider } from "../contexts/AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";

const AddMemberContent = () => {
    const { groupId, loading } = useAddMember();

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title="Add members"
                    backTo={groupId ? `/group/${groupId}` : "/"}
                    backLabel="Back to group"
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
