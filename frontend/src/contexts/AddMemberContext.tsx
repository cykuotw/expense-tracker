import { useState, useEffect, ReactNode, FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { API_URL } from "../configs/config";
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
            setEmailFeedback("");
            return;
        }

        if (!isValidEmail(debouncedEmail)) {
            setEmailFeedback("* invalid email format (example@youremail.com)");
            return;
        }

        const checkEmailValid = async () => {
            let emailExist: boolean | null = null;

            setLoading(true);
            try {
                const response = await fetch(`${API_URL}/checkEmail`, {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    credentials: "include",
                    body: JSON.stringify({ email: debouncedEmail }),
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
                    `${API_URL}/userInfo?email=${debouncedEmail}`,
                    {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        credentials: "include",
                        body: JSON.stringify({
                            email: debouncedEmail,
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
                emailFeedback,
                newMember,
                handleSubmitRelatedUsers,
                handleAddNewMember,
            }}
        >
            {children}
        </AddMemberContext.Provider>
    );
};
