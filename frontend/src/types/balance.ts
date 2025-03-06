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
