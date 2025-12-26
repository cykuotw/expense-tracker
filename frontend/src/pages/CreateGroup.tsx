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
        feedback,
        dataOk,
        createGroup,
    } = useCreateGroup();

    return (
        <div className="flex justify-center items-center py-5 h-screen md:h-auto">
            <form
                className="flex flex-col justify-center items-center max-w-md gap-y-3"
                onSubmit={createGroup}
            >
                <div className="flex flex-col py-5 text-3xl">
                    Create New Group
                </div>
                <label className="floating-label w-full">
                    <span>Group Name</span>
                    <input
                        type="text"
                        className="grow input input-bordered flex items-center w-full"
                        placeholder="Group Name"
                        value={groupName}
                        onChange={(e) => setGroupName(e.target.value)}
                    />
                </label>
                <label className="floating-label w-full">
                    <span>Group Description</span>
                    <input
                        type="text"
                        className="grow input input-bordered flex items-center w-full"
                        placeholder="Group Description (optional)"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                    />
                </label>
                <label className="floating-label w-full">
                    <span>Currency</span>
                    <select
                        className="select select-bordered w-full"
                        value={currency}
                        onChange={(e) => setCurrency(e.target.value)}
                    >
                        <option value="" disabled={true}>
                            Select Currency
                        </option>
                        <option value="CAD">CAD</option>
                        <option value="USD">USD</option>
                        <option value="NTD">NTD</option>
                    </select>
                </label>
                <button
                    type="submit"
                    className="btn btn-active btn-neutral text-lg font-light w-full"
                    disabled={!dataOk}
                >
                    Create Group
                </button>
                <div
                    className={`${
                        indicator ? "" : "hidden"
                    } flex justify-center items-center w-full`}
                >
                    <span className="loading loading-spinner loading-md"></span>
                </div>
                <div
                    className={`${feedback.length !== 0 ? "hidden" : ""}`}
                ></div>
            </form>
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
