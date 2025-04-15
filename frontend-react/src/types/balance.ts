export interface Balance {
    id: string;
    senderUserId: string;
    senderUsername: string;
    receiverUserId: string;
    receiverUsername: string;
    balance: string;
}

export interface BalanceData {
    currency: string;
    currentUser: string;
    balances: Balance[];
}

export enum SplitOption {
    Equally = "Equally",
    Unequally = "Unequally",
    YouHalf = "You-Half",
    YouFull = "You-Full",
    OtherHalf = "Other-Half",
    OtherFull = "Other-Full",
}
