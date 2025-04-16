export interface LedgerData {
    id: string;
    lenderUserId: string;
    lenderUsername: string;
    borrowerUserId: string;
    borrowerUsername: string;
    share: string;
}

export interface LedgerCreateData {
    lenderUserId: string;
    borrowerUserId: string;
    share: string;
}
