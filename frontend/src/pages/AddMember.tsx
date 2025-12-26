import { AddMemberProvider } from "../contexts/AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";

const AddMemberContent = () => {
    const {
        groupId,
        feedback,
        loading,
        relatedUserList,
        email,
        setEmail,
        emailFeedback,
        newMember,
        handleSubmitRelatedUsers,
        handleAddNewMember,
    } = useAddMember();

    return (
        <div className="flex flex-col justify-center items-center py-5 h-screen md:h-auto">
            <form
                className="flex flex-col justify-center items-center"
                onSubmit={handleSubmitRelatedUsers}
            >
                <div className="flex flex-col py-5 text-3xl">
                    Add Group Member
                </div>
                <div className="flex flex-col py-2 text-lg">
                    Your friends here
                </div>
                <div id="members" className="w-10/12">
                    {relatedUserList.length !== 0 ? (
                        relatedUserList.map((user) => {
                            return (
                                <label
                                    className="input input-ghost"
                                    key={user.userId}
                                >
                                    <input
                                        type="checkbox"
                                        defaultChecked={user.existInGroup}
                                        className="checkbox checkbox-lg"
                                        name="candidate[]"
                                        value={user.userId}
                                    />
                                    <span>{user.username}</span>
                                </label>
                            );
                        })
                    ) : (
                        <div className="text-lg">No friends found</div>
                    )}
                </div>
                <div className="w-full py-5">
                    <button
                        type="submit"
                        className="btn btn-active btn-neutral btn-wide text-lg font-light"
                    >
                        Add Members
                    </button>
                </div>
                <div
                    className={`flex justify-center items-center w-full ${
                        loading ? "" : "hidden"
                    }`}
                >
                    <span className="loading loading-spinner loading-md"></span>
                </div>
            </form>

            <div className="flex flex-col justify-center items-center space-y-2">
                <div className="flex flex-col text-lg">or a new friend</div>
                <div className="w-full justify-center items-center">
                    <input
                        type="email"
                        name="email"
                        className="input input-bordered w-full text-center bg-base-100"
                        placeholder="example@your.email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                    />
                </div>
                <div
                    className={`text-xs w-full text-center text-red-500 py-1 ${
                        emailFeedback ? "" : "hidden"
                    }`}
                >
                    {emailFeedback}
                </div>
                <div>
                    <button
                        className={`btn btn-neutral ${
                            newMember ? "" : "btn-disabled"
                        }`}
                        onClick={() => {
                            handleAddNewMember();
                        }}
                    >
                        Add
                    </button>
                </div>
            </div>

            <dialog
                id="feedback"
                className="modal"
                onClose={() => {
                    window.location.href = `/group/${groupId}`;
                }}
            >
                <div className="modal-box flex flex-col items-center">
                    <h3 className="font-bold text-lg">{feedback}</h3>
                    <p className="py-4">
                        Press ESC key or click outside to close
                    </p>
                </div>
                <form method="dialog" className="modal-backdrop">
                    <button>close</button>
                </form>
            </dialog>
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
