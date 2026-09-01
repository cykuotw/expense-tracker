import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";
import { CreateGroupProvider } from "../contexts/CreateGroupContext";
import { useCreateGroup } from "../hooks/CreateGroupContextHooks";
import { GroupTypePicker } from "../components/group/GroupTypePicker";
import MobilePageHeader from "../components/MobilePageHeader";

const CreateGroupContent = () => {
    const {
        groupName,
        setGroupName,
        description,
        setDescription,
        currency,
        setCurrency,
        groupType,
        setGroupType,
        indicator,
        dataOk,
        createGroup,
    } = useCreateGroup();

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-4xl">
                <MobilePageHeader
                    title="Create group"
                    backTo="/"
                    backLabel="Back to groups"
                    action={
                        indicator ? (
                            <span className="ui-spinner ui-spinner-sm" role="status" aria-label="Creating group" />
                        ) : (
                            <button
                                type="submit"
                                form="create-group-form"
                                className="ui-button ui-button-primary min-h-12 min-w-12 px-3"
                                aria-label="Create group"
                                disabled={!dataOk}
                            >
                                <Icon path={mdiCheckBold} size={1} aria-hidden="true" />
                            </button>
                        )
                    }
                />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Groups</div>
                        <h1 className="page-title">Create a new group</h1>
                        <p className="page-copy">
                            Set a name, choose its type, and pick the currency your group will use.
                        </p>
                    </div>
                </div>

                <form
                    id="create-group-form"
                    className="panel-card rounded-[2rem] p-4 md:p-8"
                    onSubmit={createGroup}
                >
                        <div className="grid gap-3 md:gap-5">
                            <div>
                                <div className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">Group type</div>
                                <div className="mt-2">
                                    <GroupTypePicker value={groupType} onChange={setGroupType} />
                                </div>
                            </div>
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Group name
                                </label>
                                <label className="ui-input-shell mt-2 flex items-center w-full bg-background">
                                    <input
                                        type="text"
                                        className="grow"
                                        placeholder="Group Name"
                                        value={groupName}
                                        onChange={(e) =>
                                            setGroupName(e.target.value)
                                        }
                                    />
                                </label>
                            </div>
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Description
                                </label>
                                <label className="ui-input-shell mt-2 flex items-center w-full bg-background">
                                    <input
                                        type="text"
                                        className="grow"
                                        placeholder="Group Description (optional)"
                                        value={description}
                                        onChange={(e) =>
                                            setDescription(e.target.value)
                                        }
                                    />
                                </label>
                            </div>
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Currency
                                </label>
                                <select
                                    className="ui-select mt-2 w-full"
                                    value={currency}
                                    onChange={(e) =>
                                        setCurrency(e.target.value)
                                    }
                                >
                                    <option value="" disabled={true}>
                                        Select Currency
                                    </option>
                                    <option value="CAD">CAD</option>
                                    <option value="USD">USD</option>
                                    <option value="NTD">NTD</option>
                                </select>
                            </div>
                        </div>

                        <div className="mt-8 hidden flex-col gap-3 md:flex md:flex-row md:items-center md:justify-between">
                            <button
                                type="submit"
                                className="ui-button ui-button-primary w-full sm:w-auto"
                                disabled={!dataOk}
                            >
                                Create Group
                            </button>
                            {indicator && (
                                <span className="ui-spinner ui-spinner-sm" role="status" aria-label="Creating group"></span>
                            )}
                        </div>
                </form>
            </div>
        </div>
    );
};

const CreateGroup = () => {
    return (
        <CreateGroupProvider>
            <CreateGroupContent />
        </CreateGroupProvider>
    );
};

export default CreateGroup;
