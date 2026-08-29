import { useState, useEffect, ReactNode } from "react";
import { apiFetch, asArray } from "../lib/api";
import { GroupCardData } from "../types/group";
import { HomeContext } from "./HomeContextHooks";

export const HomeProvider = ({ children }: { children: ReactNode }) => {
    const [groupCards, setGroupCards] = useState<GroupCardData[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchGroups = async () => {
            try {
                const response = await apiFetch("/groups", {
                    method: "GET",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const groups = await response.json();
                setGroupCards(asArray<GroupCardData>(groups));
            } catch (error) {
                console.log(error);
            } finally {
                setLoading(false);
            }
        };

        fetchGroups();
    }, []);

    return (
        <HomeContext.Provider value={{ groupCards, loading }}>
            {children}
        </HomeContext.Provider>
    );
};
