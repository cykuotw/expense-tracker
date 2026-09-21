import { useAddMember } from "../../hooks/AddMemberContextHooks";

interface GroupMemberManagerProps {
    creationMode?: boolean;
}

export function GroupMemberManager({
    creationMode = false,
}: GroupMemberManagerProps) {
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
        <div className="grid gap-4 lg:grid-cols-5 lg:gap-6">
            <form
                id="member-selection-form"
                className="panel-card rounded-[2rem] p-4 md:p-6 lg:col-span-3"
                onSubmit={handleSubmitRelatedUsers}
            >
                <div className="text-sm font-semibold uppercase tracking-[0.2em] text-foreground/60">
                    Existing friends
                </div>
                <div id="member-candidates" className="mt-4 space-y-2">
                    {relatedUserList.length !== 0 ? (
                        relatedUserList.map((user) => (
                            <label
                                className="grid min-h-12 cursor-pointer grid-cols-[auto_minmax(0,1fr)] items-center gap-3 rounded-2xl border border-border bg-background px-4 py-3"
                                key={user.userId}
                            >
                                <input
                                    type="checkbox"
                                    defaultChecked={user.existInGroup}
                                    className="ui-checkbox"
                                    name="candidate[]"
                                    value={user.userId}
                                    form={creationMode ? "create-group-form" : undefined}
                                />
                                <span className="min-w-0">
                                    <span className="block text-sm font-medium leading-5 text-foreground">
                                        {user.username}
                                    </span>
                                    {" "}
                                    <span className="block [overflow-wrap:anywhere] text-sm leading-5 text-foreground/65">
                                        {user.email}
                                    </span>
                                </span>
                            </label>
                        ))
                    ) : (
                        <div className="rounded-2xl border border-border bg-background p-4 text-sm text-foreground/70">
                            No friends found
                        </div>
                    )}
                </div>
                {creationMode ? (
                    <p className="mt-6 text-sm leading-6 text-foreground/65">
                        Selected members will be added when you create the group.
                    </p>
                ) : (
                    <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <button
                            type="submit"
                            className="ui-button ui-button-primary w-full sm:w-auto"
                            disabled={loading}
                        >
                            {loading ? "Updating…" : "Update members"}
                        </button>
                        {loading ? (
                            <span
                                className="ui-spinner ui-spinner-sm"
                                role="status"
                                aria-label="Updating members"
                            />
                        ) : null}
                    </div>
                )}
            </form>

            <div className="panel-card rounded-[2rem] p-4 md:p-6 lg:col-span-2">
                <div className="text-sm font-semibold uppercase tracking-[0.2em] text-foreground/60">
                    Invite by email
                </div>
                <div className="mt-4 space-y-4">
                    <label className="ui-input-shell flex w-full items-center gap-2 bg-background">
                        <span className="sr-only">Registered user email</span>
                        <input
                            type="email"
                            name="email"
                            className="grow"
                            placeholder="example@your.email"
                            value={email}
                            onChange={(event) => setEmail(event.target.value)}
                        />
                    </label>
                    <button
                        type="button"
                        className={`ui-button ui-button-primary w-full ${
                            newMember ? "" : "ui-button-disabled"
                        }`}
                        onClick={handleAddNewMember}
                        disabled={!newMember}
                    >
                        Add to selection
                    </button>
                    <p className="text-xs text-foreground/60">
                        {creationMode
                            ? "Registered users can be added now and saved when you create the group."
                            : "Registered users can be added to the selection, then saved with Update members."}
                    </p>
                </div>
            </div>
        </div>
    );
}
