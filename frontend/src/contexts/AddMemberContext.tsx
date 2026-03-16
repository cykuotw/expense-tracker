import { useState, useEffect, ReactNode, FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { isEmail } from "validator";
import { apiFetch } from "../lib/api";
import { RelatedUser } from "../types/group";
import { UserData } from "../types/user";
import useDebounce from "../hooks/useDebounce";
import { AddMemberContext } from "../hooks/AddMemberContextHooks";

interface UpdateGroupMemberPayload {
    action: "add" | "delete";
    userId: string;
    groupId: string;
}

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
                setRelatedUserList(data);
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
            await Promise.all(
                payloads.map(async (payload) => {
                    const response = await apiFetch("/group_member", {
                        method: "PUT",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        body: JSON.stringify(payload),
                    });

                    const data = await response.json();
                    if (!response.ok)
                        throw new Error(data.message || "Update failed");
                })
            );

            toast.success("Update successful!", { duration: 1000 });
            if (groupId) {
                window.setTimeout(() => {
                    navigate(`/group/${groupId}`);
                }, 1000);
            }
        } catch (error) {
            toast.error((error as Error).message);
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
            let emailExist: boolean | null = null;

            setLoading(true);
            try {
                const response = await apiFetch(
                    "/checkEmail",
                    {
                        method: "POST",
                        body: JSON.stringify({ email: debouncedEmail }),
                    },
                    { authMode: "none" }
                );
                const data = await response.json();
                emailExist = Boolean(data.exist);
            } catch (error) {
                toast.error(`Error checking email: ${error}`, {
                    id: "email-validation",
                });
                setNewMember(null);
            } finally {
                setLoading(false);
            }

            if (emailExist === null) {
                setNewMember(null);
                return;
            } else if (!emailExist) {
                toast.error("Email not found. Please contact admin.", {
                    id: "email-validation",
                });
                setNewMember(null);
                return;
            }

            try {
                const response = await apiFetch(
                    `/userInfo?email=${debouncedEmail}`,
                    {
                        method: "POST",
                        body: JSON.stringify({
                            email: debouncedEmail,
                        }),
                    },
                    { authMode: "none" }
                );
                const data = (await response.json()) as UserData;
                if (!response.ok) {
                    throw new Error("Something went wrong");
                }

                if (relatedUserList.some((user) => user.userId === data.id)) {
                    toast.error("User already in the group", {
                        id: "email-validation",
                    });
                    setNewMember(null);
                    return;
                }

                setNewMember(data);
            } catch (error) {
                console.log(error);
                setNewMember(null);
            } finally {
                setLoading(false);
            }
        };

        checkEmailValid();
    }, [debouncedEmail, relatedUserList]);

    const handleAddNewMember = () => {
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
