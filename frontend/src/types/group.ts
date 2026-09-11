export type GroupBalanceStatus = "settled" | "owed" | "owing";

export interface GroupCardData {
    id: string;
    groupName: string;
    description: string;
    currency: string;
    groupType: string;
    balanceStatus: GroupBalanceStatus;
    balanceAmount: string;
}

export interface GroupMember {
    userId: string;
    username: string;
}

export type GroupMembersLoadStatus = "idle" | "loading" | "ready" | "error";

export interface GroupInfo {
    currentUserId: string;
    groupName: string;
    description: string;
    currency: string;
    currencyEditable: boolean;
    detailsEditable: boolean;
    groupType: string;
    members: GroupMember[];
}

export interface RelatedUser {
    userId: string;
    username: string;
    existInGroup: boolean;
}

export interface GroupListItem {
    id: string;
    groupName: string;
    description: string;
    currency: string;
    groupType: string;
}

export interface GroupNewData {
    groupName: string;
    description: string;
    currency: string;
    groupType: string;
}
