import { useState, useEffect, ReactNode } from "react";
import { apiFetch, asArray } from "../lib/api";
import { GroupCardData } from "../types/group";
import { HomeContext } from "./HomeContextHooks";

export const HomeProvider = ({ children }: { children: ReactNode }) => {
    const [groupCards, setGroupCards] = useState<GroupCardData[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const abortController = new AbortController();
        let active = true;

        const fetchGroups = async () => {
            try {
                const response = await apiFetch("/groups", {
                    method: "GET",
                    signal: abortController.signal,
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const groups = await response.json();
                if (active) {
                    setGroupCards(asArray<GroupCardData>(groups));
                }
            } catch (error) {
                if (active && !abortController.signal.aborted) {
                    console.log(error);
                }
            } finally {
                if (active) {
                    setLoading(false);
                }
            }
        };

        void fetchGroups();
        return () => {
            active = false;
            abortController.abort();
        };
    }, []);

    return (
        <HomeContext.Provider value={{ groupCards, loading }}>
            {children}
        </HomeContext.Provider>
    );
};
