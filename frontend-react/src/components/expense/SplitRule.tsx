import { useEffect, useState } from "react";

import { GroupMember } from "../../types/group";
import { API_URL } from "../../configs/config";

interface SplitRuleProps {
    groupId: string;
}

const SplitRule = ({ groupId }: SplitRuleProps) => {
    // handle fetch group member on load
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);

    useEffect(() => {
        const fetchGroupMembers = async () => {
            const response = await fetch(`${API_URL}/group_member/${groupId}`, {
                method: "GET",
                credentials: "include",
            });
            if (response.ok) {
                const data: GroupMember[] = await response.json();
                setGroupMembers(data);
            }
        };

        fetchGroupMembers();
    }, []);

    if (groupMembers.length === 0) {
        return <></>;
    }

    return <>split rule</>;
};

export default SplitRule;
