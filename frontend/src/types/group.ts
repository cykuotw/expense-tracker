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

export interface RelatedUser {
    userId: string;
    username: string;
    existInGroup: boolean;
}
