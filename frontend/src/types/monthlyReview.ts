export type MonthlyReviewState =
    | "not_closed"
    | "unpublished"
    | "empty"
    | "published";

export interface MonthlyReviewAmount {
    name: string;
    amount: string;
}

export interface MonthlyReviewUserAmount {
    userId: string;
    username: string;
    amount: string;
}

export interface MonthlyReviewExpense {
    id: string;
    description: string;
    occurredOn: string;
    expenseType: string;
    category: string;
    payerId: string;
    payerName: string;
    total: string;
    settled: boolean;
}

export interface MonthlyReviewCurrency {
    currency: string;
    total: string;
    expenseCount: number;
    categories: MonthlyReviewAmount[];
    payers: MonthlyReviewUserAmount[];
    memberNetTotals: MonthlyReviewUserAmount[];
}

export interface MonthlyReviewExpensePage {
    expenses: MonthlyReviewExpense[];
    nextCursor?: string;
}

export interface MonthlyReviewData {
    groupId: string;
    groupName: string;
    month: string;
    state: MonthlyReviewState;
    publishedAt?: string;
    currencies: MonthlyReviewCurrency[];
}

export interface MonthlyReviewTrendMonth {
    month: string;
    total: string;
    expenseCount: number;
}

export interface MonthlyReviewTrendCurrency {
    currency: string;
    months: MonthlyReviewTrendMonth[];
}

export interface MonthlyReviewTrendData {
    groupId: string;
    groupName: string;
    startMonth: string;
    endMonth: string;
    latestReportMonth?: string;
    currencies: MonthlyReviewTrendCurrency[];
}
