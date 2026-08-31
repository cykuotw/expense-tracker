import {
    useState,
    useEffect,
    useRef,
    ReactNode,
    FormEvent,
    SetStateAction,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { isEmail } from "validator";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { RelatedUser } from "../types/group";
import { UserData } from "../types/user";
import useDebounce from "../hooks/useDebounce";
import { AddMemberContext } from "../hooks/AddMemberContextHooks";

const UPDATE_MEMBERS_FALLBACK = "Update failed";
const CHECK_EMAIL_FALLBACK = "Unable to check email";
const LOOKUP_USER_FALLBACK = "Unable to look up user";

export const AddMemberProvider = ({ children }: { children: ReactNode }) => {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");

    const [savingMembers, setSavingMembers] = useState(false);
    const [checkingEmail, setCheckingEmail] = useState(false);
    const [relatedUserList, setRelatedUserList] = useState<RelatedUser[]>([]);
    const relatedUserListRef = useRef<RelatedUser[]>([]);
    const lookupGenerationRef = useRef(0);

    const [email, setEmail] = useState("");
    const debouncedEmail = useDebounce(email, 300);
    const [newMember, setNewMember] = useState<UserData | null>(null);
    const loading = savingMembers || checkingEmail;

    const invalidateEmailLookup = () => {
        lookupGenerationRef.current += 1;
        setCheckingEmail(false);
    };

    const updateEmail = (nextEmail: SetStateAction<string>) => {
        invalidateEmailLookup();
        setEmail(nextEmail);
    };

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

    useEffect(() => {
        relatedUserListRef.current = relatedUserList;
    }, [relatedUserList]);

    const handleSubmitRelatedUsers = async (e: FormEvent) => {
        e.preventDefault();
        setSavingMembers(true);

        const formData = new FormData(e.currentTarget as HTMLFormElement);
        const selectedUserIds = new Set(
            formData.getAll("candidate[]") as string[]
        );
        const memberIds = relatedUserList
            .filter((user) => selectedUserIds.has(user.userId))
            .map((user) => user.userId);

        try {
            const response = await apiFetch("/group_members", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ groupId, memberIds }),
            });
            if (!response.ok) {
                toast.error(await getResponseErrorMessage(response, UPDATE_MEMBERS_FALLBACK));
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
            setSavingMembers(false);
        }
    };

    useEffect(() => {
        const lookupGeneration = ++lookupGenerationRef.current;
        const isCurrentLookup = () =>
            lookupGenerationRef.current === lookupGeneration;

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
            setCheckingEmail(true);
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
                    const errorMessage = await getResponseErrorMessage(
                        response,
                        CHECK_EMAIL_FALLBACK
                    );
                    if (!isCurrentLookup()) return;
                    toast.error(
                        errorMessage,
                        { id: "email-validation" }
                    );
                    setNewMember(null);
                    return;
                }

                const data = (await response.json()) as { exist?: boolean };
                if (!isCurrentLookup()) return;
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
                    const errorMessage = await getResponseErrorMessage(
                        userResponse,
                        LOOKUP_USER_FALLBACK
                    );
                    if (!isCurrentLookup()) return;
                    toast.error(
                        errorMessage,
                        { id: "email-validation" }
                    );
                    setNewMember(null);
                    return;
                }

                const userData = (await userResponse.json()) as UserData;
                if (!isCurrentLookup()) return;

                if (
                    relatedUserListRef.current.some(
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
                if (!isCurrentLookup()) return;
                toast.error(requestFallback, { id: "email-validation" });
                setNewMember(null);
            } finally {
                if (isCurrentLookup()) {
                    setCheckingEmail(false);
                }
            }
        };

        checkEmailValid();
    }, [debouncedEmail]);

    const handleAddNewMember = () => {
        if (!newMember) return;

        invalidateEmailLookup();
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
        toast.success("Member added", {
            id: "email-validation",
            duration: 1000,
        });
    };

    return (
        <AddMemberContext.Provider
            value={{
                groupId,
                loading,
                relatedUserList,
                email,
                setEmail: updateEmail,
                newMember,
                handleSubmitRelatedUsers,
                handleAddNewMember,
            }}
        >
            {children}
        </AddMemberContext.Provider>
    );
};
