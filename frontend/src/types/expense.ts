import { ItemCreateData, ItemData } from "./item";
import { LedgerCreateData, LedgerData, LedgerUpdateData } from "./ledger";

export interface ExpenseData {
    expenseId: string;
    description: string;
    total: string;
    expenseTime: string;
    currentUser: string;
    currency: string;
    isSettled: boolean;
    payerUserIds: string[];
    payerUsernames: string[];
}

export interface ExpenseDetailData {
    expenseId: string;
    description: string;
    createdByUserID: string;
    createdByUsername: string;
    expenseTypeId: string;
    expenseType: string;
    subTotal: string;
    taxFeeTip: string;
    total: string;
    currency: string;
    expenseTime: string;
    invoiceUrl: string;
    currentUser: string;
    groupId: string;
    splitRule: string;
    items: ItemData[];
    ledgers: LedgerData[];
}

export interface ExpenseTypeItem {
    id: string;
    category: string;
    name: string;
}

export interface ExpenseCreateData {
    description: string;
    groupId: string;
    payByUserId: string;
    expTypeId: string;
    total: string;
    currency: string;
    splitRule: string;
    ledgers: LedgerCreateData[];

    createByUserId?: string;

    providerName?: string;
    subTotal?: string;
    taxFeeTip?: string;
    invoiceUrl?: string;
    items?: ItemCreateData[];
}

export interface ExpenseUpdateData
    extends Omit<ExpenseCreateData, "createByUserId" | "ledgers"> {
    ledgers: LedgerUpdateData[];
}
