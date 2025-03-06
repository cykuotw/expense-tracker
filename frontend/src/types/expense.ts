export interface ExpenseData {
    expenseId: string;
    description: string;
    total: string;
    expenseTime: string;
    currentUser: string;
    currency: string;
    payerUserIds: string[];
    payerUsernames: string[];
}
