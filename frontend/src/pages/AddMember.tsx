import { useSearchParams } from "react-router-dom";
import { useEffect, useState } from "react";

import { API_URL } from "../configs/config";
import { RelatedUser } from "../types/group";
import useDebounce from "../hooks/useDebounce";
import { UserData } from "../types/user";

interface UpdateGroupMemberPayload {
    action: "add" | "delete";
    userId: string;
    groupId: string;
}

export default function AddMember() {
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");

    const [feedback, setFeedback] = useState("");
    const [loading, setLoading] = useState(false);
    const [relatedUserList, setRelatedUserList] = useState<RelatedUser[]>([]);

    const [email, setEmail] = useState("");
    const [emailFeedback, setEmailFeedback] = useState("");
    const debouncedEmail = useDebounce(email, 300);
    const [newMember, setNewMember] = useState<UserData | null>(null);
    const isValidEmail = (email: string) => {
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    };

    useEffect(() => {
        const fetchRelatedUsers = async () => {
            try {
                const response = await fetch(
                    `${API_URL}/related_member?g=${groupId}`,
                    {
                        method: "GET",
                        credentials: "include",
                        headers: {
                            "Content-Type": "application/json",
                        },
                    }
                );
                const data = await response.json();
                setRelatedUserList(data);
            } catch (error) {
                console.log(error);
            }
        };

        fetchRelatedUsers();
    }, []);

    const handleSubmitRelatedUsers = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setFeedback("");

        const formData = new FormData(e.currentTarget as HTMLFormElement);
        const selectedUserIds = new Set(
            formData.getAll("candidate[]") as string[]
        );
        const payloads: UpdateGroupMemberPayload[] = relatedUserList.map(
            (user) => ({
                action: selectedUserIds.has(user.userId) ? "add" : "delete",
                userId: user.userId,
                groupId: groupId as string,
            })
        );

        try {
            payloads.forEach(async (payload) => {
                const response = await fetch(`${API_URL}/group_member`, {
                    method: "PUT",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify(payload),
                });

                const data = await response.json();
                if (!response.ok)
                    throw new Error(data.message || "Update failed");
            });

            setFeedback("✅ Update successful!");
            (
                document.getElementById("feedback") as HTMLDialogElement
            ).showModal();
        } catch (error) {
            setFeedback(`❌ ${(error as Error).message}`);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (!debouncedEmail) {
            setEmailFeedback("");
            return;
        }

        if (!isValidEmail(debouncedEmail)) {
            setEmailFeedback("* invalid email format (example@youremail.com)");
            return;
        }

        const checkEmailValid = async () => {
            if (!email) {
                return;
            }

            let emailExist: boolean | null = null;

            setLoading(true);
            try {
                const response = await fetch(`${API_URL}/checkEmail`, {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    credentials: "include",
                    body: JSON.stringify({ email: email }),
                });
                const data = await response.json();
                if (data.exist) {
                    emailExist = true;
                }
            } catch (error) {
                setEmailFeedback(`Error checking email: ${error}`);
            } finally {
                setLoading(false);
            }

            if (emailExist === null) {
                return;
            } else if (!emailExist) {
                setEmailFeedback("* email not found");
                return;
            }

            try {
                const response = await fetch(
                    `${API_URL}/userInfo?email=${email}`,
                    {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        credentials: "include",
                        body: JSON.stringify({
                            email: email,
                        }),
                    }
                );
                const data = (await response.json()) as UserData;
                if (!response.ok) {
                    throw new Error("Something went wrong");
                }

                if (relatedUserList.some((user) => user.userId === data.id)) {
                    setEmailFeedback("* user already in the group");
                    return;
                }

                setNewMember(data);

                setEmailFeedback("");
            } catch (error) {
                console.log(error);
            } finally {
                setLoading(false);
            }
        };

        checkEmailValid();
    }, [debouncedEmail]);

    const handleAddNewMember = async () => {
        if (!newMember) return;

        const updatedUserList = [
            ...relatedUserList,
            {
                userId: newMember.id,
                username: newMember.username,
                existInGroup: true,
            },
        ];
        setRelatedUserList(updatedUserList);
        setEmail("");
        setNewMember(null);
    };

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
}
