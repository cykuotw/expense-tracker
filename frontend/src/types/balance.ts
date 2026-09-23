export interface Balance {
    id: string;
    senderUserId: string;
    senderUsername: string;
    receiverUserId: string;
    receiverUsername: string;
    balance: string;
    currency: string;
}

export interface SettlementPreviewContribution {
    balanceId: string;
    sourceCurrency: string;
    sourceAmount: string;
    rate: string | null;
    previewAmount: string | null;
}

export interface SettlementPreview {
    currency: string;
    complete: boolean;
    netAmount: string | null;
    contributions: SettlementPreviewContribution[];
    missingRateCurrencies: string[];
}

export interface BalanceData {
    currency: string;
    currentUser: string;
    balances: Balance[];
    settlementPreview: SettlementPreview | null;
}

export enum SplitOption {
    Equally = "Equally",
    Unequally = "Unequally",
    YouHalf = "You-Half",
    YouFull = "You-Full",
    OtherHalf = "Other-Half",
    OtherFull = "Other-Full",
}
