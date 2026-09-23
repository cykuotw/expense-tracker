export type GroupBalanceStatus =
    | "settled"
    | "owed"
    | "owing"
    | "preview_unavailable";

export interface GroupCurrencySetting {
    currency: string;
    enabledForNewExpenses: boolean;
    historical: boolean;
    hasCurrentBalance: boolean;
    previewRate: string | null;
}

export interface GroupCurrencySettings {
    settlementPreviewCurrency: string;
    currencies: GroupCurrencySetting[];
}

export interface GroupCardData {
    id: string;
    groupName: string;
    description: string;
    currency: string;
    groupType: string;
    balanceStatus: GroupBalanceStatus;
    balanceAmount: string;
    settlementPreviewComplete?: boolean;
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
    currencySettings: GroupCurrencySettings;
    currencyEditable: boolean;
    detailsEditable: boolean;
    groupType: string;
    members: GroupMember[];
}

export interface RelatedUser {
    userId: string;
    username: string;
    email: string;
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
    memberIds: string[];
    currencySettings: GroupCurrencySettings;
}
