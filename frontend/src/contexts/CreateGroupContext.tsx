import { useState, ReactNode, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, getResponseErrorMessage } from "../lib/api";
import { GroupNewData } from "../types/group";
import { CreateGroupContext } from "../hooks/CreateGroupContextHooks";

export const CreateGroupProvider = ({ children }: { children: ReactNode }) => {
    const navigate = useNavigate();
    const [indicator, setIndicator] = useState<boolean>(false);
    const [createdGroupId, setCreatedGroupId] = useState<string | null>(null);

    const [groupName, setGroupName] = useState<string>("");
    const [description, setDescription] = useState<string>("");
    const [currency, setCurrency] = useState<string>("");
    const [groupType, setGroupType] = useState<string>("home");

    const dataOk =
        createdGroupId === null &&
        groupName.trim().length > 0 &&
        currency.length > 0;

    const createGroup = async (e: FormEvent) => {
        e.preventDefault();
        if (createdGroupId) return;

        const formData = new FormData(e.currentTarget as HTMLFormElement);
        const memberIds = formData.getAll("candidate[]") as string[];

        const payload: GroupNewData = {
            groupName: groupName,
            description: description,
            currency: currency,
            groupType,
            memberIds,
        };

        try {
            setIndicator(true);

            const response = await apiFetch("/create_group", {
                method: "POST",
                body: JSON.stringify(payload),
            });

            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        "Failed to create group"
                    )
                );
                return;
            }

            const data = (await response.json()) as { groupId?: unknown };
            if (typeof data.groupId === "string" && data.groupId) {
                toast.success("Group created", { duration: 1000 });
                setCreatedGroupId(data.groupId);
            } else {
                toast.error("Group created, but member setup is unavailable");
                navigate("/");
            }
        } catch (err) {
            toast.error("Failed to create group");
            console.error("Error creating group:", err);
        } finally {
            setIndicator(false);
        }
    };

    return (
        <CreateGroupContext.Provider
            value={{
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
            }}
        >
            {children}
        </CreateGroupContext.Provider>
    );
};
