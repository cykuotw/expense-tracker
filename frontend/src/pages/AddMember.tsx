import { AddMemberProvider } from "../contexts/AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";

const AddMemberContent = () => {
    const {
        loading,
        relatedUserList,
        email,
        setEmail,
        newMember,
        handleSubmitRelatedUsers,
        handleAddNewMember,
    } = useAddMember();

    return (
        <div className="page-shell">
            <div className="page-container max-w-5xl">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group Members</div>
                        <h1 className="page-title">Add members</h1>
                        <p className="page-copy">
                            Add existing friends or invite a new person by
                            email.
                        </p>
                    </div>
                </div>

                <div className="grid gap-6 lg:grid-cols-5">
                        <form
                            className="panel-card rounded-[2rem] p-6 lg:col-span-3"
                            onSubmit={handleSubmitRelatedUsers}
                        >
                            <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                Existing friends
                            </div>
                            <div id="members" className="mt-4 space-y-2">
                                {relatedUserList.length !== 0 ? (
                                    relatedUserList.map((user) => {
                                        return (
                                            <label
                                                className="flex items-center gap-3 rounded-2xl border border-base-200 bg-base-100 px-4 py-3"
                                                key={user.userId}
                                            >
                                                <input
                                                    type="checkbox"
                                                    defaultChecked={
                                                        user.existInGroup
                                                    }
                                                    className="checkbox checkbox-md"
                                                    name="candidate[]"
                                                    value={user.userId}
                                                />
                                                <span className="text-sm font-medium">
                                                    {user.username}
                                                </span>
                                            </label>
                                        );
                                    })
                                ) : (
                                    <div className="rounded-2xl border border-base-200 bg-base-100 p-4 text-sm text-base-content/70">
                                        No friends found
                                    </div>
                                )}
                            </div>
                            <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                                <button
                                    type="submit"
                                    className="btn btn-neutral w-full sm:w-auto"
                                >
                                    Update Members
                                </button>
                                {loading && (
                                    <span className="loading loading-spinner loading-md"></span>
                                )}
                            </div>
                        </form>

                        <div className="panel-card rounded-[2rem] p-6 lg:col-span-2">
                            <div className="text-sm font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                Invite by email
                            </div>
                            <div className="mt-4 space-y-4">
                                <label className="input input-bordered flex items-center gap-2 w-full bg-base-100">
                                    <input
                                        type="email"
                                        name="email"
                                        className="grow"
                                        placeholder="example@your.email"
                                        value={email}
                                        onChange={(e) =>
                                            setEmail(e.target.value)
                                        }
                                    />
                                </label>
                                <button
                                    className={`btn btn-neutral w-full ${
                                        newMember ? "" : "btn-disabled"
                                    }`}
                                    onClick={() => {
                                        handleAddNewMember();
                                    }}
                                >
                                    Add
                                </button>
                                <p className="text-xs text-base-content/60">
                                    We will only add users already registered.
                                </p>
                            </div>
                        </div>
                    </div>
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
