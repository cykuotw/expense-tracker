import { CreateGroupProvider } from "../contexts/CreateGroupContext";
import { useCreateGroup } from "../hooks/CreateGroupContextHooks";

const CreateGroupContent = () => {
    const {
        groupName,
        setGroupName,
        description,
        setDescription,
        currency,
        setCurrency,
        indicator,
        dataOk,
        createGroup,
    } = useCreateGroup();

    return (
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-28 md:pb-0">
            <div className="mx-auto w-full max-w-4xl px-4 py-10 md:py-14">
                <div className="flex flex-col gap-8">
                    <div className="space-y-3">
                        <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                            Groups
                        </div>
                        <h1 className="text-3xl font-semibold md:text-4xl">
                            Create a new group
                        </h1>
                        <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                            Set a name, add a short description, and pick the
                            currency your group will use.
                        </p>
                    </div>

                    <form
                        className="rounded-3xl border border-base-300 bg-base-100/90 p-6 shadow-sm"
                        onSubmit={createGroup}
                    >
                        <div className="grid gap-5">
                            <div>
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Group name
                                </label>
                                <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
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
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Description
                                </label>
                                <label className="input input-bordered mt-2 flex items-center w-full bg-base-100">
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
                                <label className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                                    Currency
                                </label>
                                <select
                                    className="select select-bordered mt-2 w-full"
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

                        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                            <button
                                type="submit"
                                className="btn btn-neutral w-full sm:w-auto"
                                disabled={!dataOk}
                            >
                                Create Group
                            </button>
                            {indicator && (
                                <span className="loading loading-spinner loading-md"></span>
                            )}
                        </div>
                    </form>
                </div>
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
