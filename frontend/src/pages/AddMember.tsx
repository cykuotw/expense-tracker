import MobilePageHeader from "../components/MobilePageHeader";
import { GroupMemberManager } from "../components/group/GroupMemberManager";
import { AddMemberProvider } from "../contexts/AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";

const AddMemberContent = () => {
    const { groupId } = useAddMember();

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title="Add members"
                    backTo={groupId ? `/group/${groupId}` : "/"}
                    backLabel="Back to group"
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
