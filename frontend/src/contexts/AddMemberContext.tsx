import { useState, useEffect, ReactNode, FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { isEmail } from "validator";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { RelatedUser } from "../types/group";
import { UserData } from "../types/user";
import useDebounce from "../hooks/useDebounce";
import { AddMemberContext } from "../hooks/AddMemberContextHooks";

interface UpdateGroupMemberPayload {
    action: "add" | "delete";
    userId: string;
    groupId: string;
}

const UPDATE_MEMBERS_FALLBACK = "Update failed";
const CHECK_EMAIL_FALLBACK = "Unable to check email";
const LOOKUP_USER_FALLBACK = "Unable to look up user";

export const AddMemberProvider = ({ children }: { children: ReactNode }) => {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");

    const [loading, setLoading] = useState(false);
    const [relatedUserList, setRelatedUserList] = useState<RelatedUser[]>([]);

    const [email, setEmail] = useState("");
    const debouncedEmail = useDebounce(email, 300);
    const [newMember, setNewMember] = useState<UserData | null>(null);

    useEffect(() => {
        const fetchRelatedUsers = async () => {
            try {
                const response = await apiFetch(`/related_member?g=${groupId}`, {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const data = await response.json();
                setRelatedUserList(asArray<RelatedUser>(data));
            } catch (error) {
                console.log(error);
            }
        };

        fetchRelatedUsers();
    }, [groupId]);

    const handleSubmitRelatedUsers = async (e: FormEvent) => {
        e.preventDefault();
        setLoading(true);

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
            const errorMessages = await Promise.all(
                payloads.map(async (payload) => {
                    const response = await apiFetch("/group_member", {
                        method: "PUT",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        body: JSON.stringify(payload),
                    });

                    if (!response.ok) {
                        return getResponseErrorMessage(
                            response,
                            UPDATE_MEMBERS_FALLBACK
                        );
                    }

                    return null;
                })
            );

            const errorMessage = errorMessages.find(
                (message): message is string => message !== null
            );
            if (errorMessage) {
                toast.error(errorMessage);
                return;
            }

            toast.success("Update successful!", { duration: 1000 });
            if (groupId) {
                window.setTimeout(() => {
                    navigate(`/group/${groupId}`);
                }, 1000);
            }
        } catch {
            toast.error(UPDATE_MEMBERS_FALLBACK);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (!debouncedEmail) {
            setNewMember(null);
            return;
        }

        if (!isEmail(debouncedEmail)) {
            toast.error("Invalid email format (example@youremail.com)", {
                id: "email-validation",
            });
            setNewMember(null);
            return;
        }

        const checkEmailValid = async () => {
            setLoading(true);
            let requestFallback = CHECK_EMAIL_FALLBACK;

            try {
                const response = await apiFetch(
                    "/checkEmail",
                    {
                        method: "POST",
                        body: JSON.stringify({ email: debouncedEmail }),
                    },
                    { authMode: "none" }
                );
                if (!response.ok) {
                    toast.error(
                        await getResponseErrorMessage(
                            response,
                            CHECK_EMAIL_FALLBACK
                        ),
                        { id: "email-validation" }
                    );
                    setNewMember(null);
                    return;
                }

                const data = (await response.json()) as { exist?: boolean };
                if (!data.exist) {
                    toast.error("Email not found. Please contact admin.", {
                        id: "email-validation",
                    });
                    setNewMember(null);
                    return;
                }

                requestFallback = LOOKUP_USER_FALLBACK;
                const userResponse = await apiFetch(
                    `/userInfo?email=${debouncedEmail}`,
                    {
                        method: "POST",
                        body: JSON.stringify({
                            email: debouncedEmail,
                        }),
                    },
                    { authMode: "none" }
                );
                if (!userResponse.ok) {
                    toast.error(
                        await getResponseErrorMessage(
                            userResponse,
                            LOOKUP_USER_FALLBACK
                        ),
                        { id: "email-validation" }
                    );
                    setNewMember(null);
                    return;
                }

                const userData = (await userResponse.json()) as UserData;

                if (
                    relatedUserList.some(
                        (user) => user.userId === userData.id
                    )
                ) {
                    toast.error("User already in the group", {
                        id: "email-validation",
                    });
                    setNewMember(null);
                    return;
                }

                setNewMember(userData);
            } catch {
                toast.error(requestFallback, { id: "email-validation" });
                setNewMember(null);
            } finally {
                setLoading(false);
            }
        };

        checkEmailValid();
    }, [debouncedEmail, relatedUserList]);

    const handleAddNewMember = () => {
        if (!newMember) return;

        setRelatedUserList((currentUsers) => [
            ...currentUsers,
            {
                userId: newMember.id,
                username: newMember.username,
                existInGroup: true,
            },
        ]);
        setEmail("");
        setNewMember(null);
    };

    return (
        <AddMemberContext.Provider
            value={{
                groupId,
                loading,
                relatedUserList,
                email,
                setEmail,
                newMember,
                handleSubmitRelatedUsers,
                handleAddNewMember,
            }}
        >
            {children}
        </AddMemberContext.Provider>
    );
};
