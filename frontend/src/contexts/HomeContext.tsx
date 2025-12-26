import { useState, useEffect, ReactNode } from "react";
import { API_URL } from "../configs/config";
import { GroupCardData } from "../types/group";
import { HomeContext } from "./HomeContextHooks";

export const HomeProvider = ({ children }: { children: ReactNode }) => {
    const [groupCards, setGroupCards] = useState<GroupCardData[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchGroups = async () => {
            try {
                const response = await fetch(`${API_URL}/groups`, {
                    method: "GET",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const groups = await response.json();
                setGroupCards(groups);
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
