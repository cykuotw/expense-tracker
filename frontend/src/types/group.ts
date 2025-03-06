export interface GroupCardData {
    id: string;
    groupName: string;
    description: string;
}

export interface GroupMember {
    userId: string;
    username: string;
}

export interface GroupInfo {
    groupName: string;
    description: string;
    currency: string;
    members: GroupMember[];
}
