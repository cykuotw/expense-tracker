import { useEffect, useState } from "react";

import { API_URL } from "../configs/config";
import GroupCard from "../components/group/GroupCard";
import { GroupCardData } from "../types/group";

export default function Home() {
    const [groupCards, setGroupCards] = useState<GroupCardData[]>([]);

    useEffect(() => {
        const fetchGroups = async () => {
            let groups: GroupCardData[] = [];
            try {
                const response = await fetch(`${API_URL}/groups`, {
                    method: "GET",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                groups = await response.json();
                setGroupCards(groups);
            } catch (error) {
                console.log(error);
            }
        };

        fetchGroups();
    }, []);

    return (
        <div className="h-screen">
            <div className="flex flex-wrap justify-center items-center py-5 md:h-auto">
                {groupCards.map((group) => (
                    <GroupCard key={group.id} {...group} />
                ))}
            </div>
            <div className="py-10 block md:hidden"></div>
        </div>
    );
}
