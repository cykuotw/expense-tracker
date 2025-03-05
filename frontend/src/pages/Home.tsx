import { useEffect, useState } from "react";
import { API_URL } from "../configs/config";
import GroupCard from "../components/group/GroupCard";

export default function Home() {
    interface GroupCardData {
        id: string;
        groupName: string;
        description: string;
    }

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
            } catch (error) {
                console.log(error);
            } finally {
                setGroupCards(groups);
            }
        };

        fetchGroups();
    }, []);

    return (
        <div className="h-screen">
            <div className="flex flex-wrap justify-center items-center py-5 md:h-auto">
                {groupCards.map((group) => (
                    <GroupCard
                        key={group.id}
                        Id={group.id}
                        GroupName={group.groupName}
                        Description={group.description}
                    />
                ))}
            </div>
            <div className="py-10 block md:hidden"></div>
        </div>
    );
}
