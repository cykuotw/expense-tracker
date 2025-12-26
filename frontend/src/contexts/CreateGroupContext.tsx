import { useState, useEffect, ReactNode, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { API_URL } from "../configs/config";
import { GroupNewData } from "../types/group";
import { CreateGroupContext } from "../hooks/CreateGroupContextHooks";

export const CreateGroupProvider = ({ children }: { children: ReactNode }) => {
    const navigate = useNavigate();
    const [indicator, setIndicator] = useState<boolean>(false);
    const [feedback, setFeedback] = useState<string>("");
    const [dataOk, setDataOk] = useState<boolean>(false);

    const [groupName, setGroupName] = useState<string>("");
    const [description, setDescription] = useState<string>("");
    const [currency, setCurrency] = useState<string>("");

    useEffect(() => {
        const ok = groupName.length > 0 && currency.length > 0;
        setDataOk(ok);
    }, [groupName, description, currency]);

    const createGroup = async (e: FormEvent) => {
        e.preventDefault();

        const payload: GroupNewData = {
            groupName: groupName,
            description: description,
            currency: currency,
        };

        try {
            setIndicator(true);
            setFeedback("");

            const response = await fetch(`${API_URL}/create_group`, {
                method: "POST",
                credentials: "include",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(payload),
            });

            if (!response.ok) {
                const errorData = await response.json();
                setFeedback(errorData.message || "Failed to create group");
                return;
            }

            navigate("/");
        } catch (err) {
            setFeedback("Failed to create group");
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
                indicator,
                feedback,
                dataOk,
                createGroup,
            }}
        >
            {children}
        </CreateGroupContext.Provider>
    );
};
