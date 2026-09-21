import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";
import { useEffect } from "react";
import { Link } from "react-router-dom";
import { CreateGroupProvider } from "../contexts/CreateGroupContext";
import { useCreateGroup } from "../hooks/CreateGroupContextHooks";
import { GroupTypePicker } from "../components/group/GroupTypePicker";
import { CurrencyPicker } from "../components/group/CurrencyPicker";
import MobilePageHeader from "../components/MobilePageHeader";
import DesktopBackLink from "../components/DesktopBackLink";
import { useCurrencies } from "../hooks/useCurrencies";
import { GroupMemberManager } from "../components/group/GroupMemberManager";
import { AddMemberProvider } from "../contexts/AddMemberContext";

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
        createdGroupId,
        indicator,
        dataOk,
        createGroup,
    } = useCreateGroup();
    const { currencies, loading: currenciesLoading, error: currenciesError, reload } = useCurrencies();

    useEffect(() => {
        if (!currenciesLoading && !currenciesError && !currency && currencies.some(({ code }) => code === "CAD")) {
            setCurrency("CAD");
        }
    }, [currencies, currenciesError, currenciesLoading, currency, setCurrency]);

    useEffect(() => {
        if (!createdGroupId) return;
        document
            .getElementById("create-group-members")
            ?.scrollIntoView?.({ block: "start" });
    }, [createdGroupId]);

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-5xl">
                <MobilePageHeader
                    title={createdGroupId ? "Manage members" : "Create group"}
                    backTo={createdGroupId ? `/group/${createdGroupId}` : "/"}
                    backLabel={createdGroupId ? "View group" : "Back to groups"}
                    action={
                        createdGroupId ? (
                            <Link
                                to={`/group/${createdGroupId}`}
                                className="ui-button ui-button-primary min-h-12 px-4"
                            >
                                Done
                            </Link>
                        ) : indicator ? (
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
                <DesktopBackLink
                    to={createdGroupId ? `/group/${createdGroupId}` : "/"}
                    label={createdGroupId ? "View group" : "Back to groups"}
                />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Groups</div>
                        <h1 className="page-title">
                            {createdGroupId
                                ? "Manage group members"
                                : "Create a new group"}
                        </h1>
                        <p className="page-copy">
                            {createdGroupId
                                ? "Your group and selected members are saved. You can make further changes below."
                                : "Set a name, choose its type, and pick the currency your group will use."}
                        </p>
                    </div>
                </div>

                {createdGroupId ? (
                    <div className="panel-card rounded-[2rem] p-4 md:p-6">
                        <div className="page-eyebrow">Group created</div>
                        <h2 className="mt-1 text-xl font-semibold text-foreground md:text-2xl">
                            {groupName.trim()}
                        </h2>
                        <p className="mt-1 text-sm leading-6 text-foreground/65">
                            The selected members were added with the group. You
                            can review them below or continue to the group.
                        </p>
                        <Link
                            to={`/group/${createdGroupId}`}
                            className="ui-button ui-button-ghost mt-4 w-full sm:w-auto"
                        >
                            View group
                        </Link>
                    </div>
                ) : (
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
                                <div className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                                    Currency
                                </div>
                                <div className="mt-2">
                                    <CurrencyPicker
                                    value={currency}
                                    currencies={currencies}
                                    onChange={setCurrency}
                                    disabled={currenciesLoading || Boolean(currenciesError)}
                                    />
                                </div>
                                {currenciesError ? (
                                    <p className="mt-2 text-sm text-destructive" role="alert">
                                        Could not load currencies. <button className="underline" type="button" onClick={() => void reload()}>Try again</button>
                                    </p>
                                ) : null}
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
                )}

                <section
                    id="create-group-members"
                    className="mt-4 scroll-mt-4 md:mt-6 md:scroll-mt-6"
                    aria-labelledby="create-group-members-heading"
                >
                    <div className="mb-3 md:mb-4">
                        <div className="page-eyebrow">Members</div>
                        <h2
                            id="create-group-members-heading"
                            className="mt-1 text-xl font-semibold text-foreground md:text-2xl"
                        >
                            Manage members
                        </h2>
                        <p className="mt-1 text-sm leading-6 text-foreground/65">
                            {createdGroupId
                                ? "Select existing friends or find a registered user by email, then update members."
                                : "Select existing friends or find a registered user by email. They will be added when you create the group."}
                        </p>
                    </div>
                    <AddMemberProvider
                        groupId={createdGroupId}
                        returnToGroupAfterSave={false}
                        allowPreCreate
                    >
                        <GroupMemberManager creationMode={!createdGroupId} />
                    </AddMemberProvider>
                </section>
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
