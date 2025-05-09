export interface ItemData {
    itemId: string;
    itemName: string;
    itemSubTotal: string;
}

export interface ItemCreateData {
    itemName: string;
    amount: string;
    unit: string;
    unitPrice: string;
}

export interface ItemUpdateData extends ItemCreateData {
    itemId: string;
}
