export interface ItemData {
    itemId: string;
    itemName: string;
    itemSubTotal: string;
    description?: string;
    quantity?: string | null;
    unit?: string | null;
    unitPrice?: string | null;
    lineTotal?: string;
    position?: number;
}

export interface ItemCreateData {
    description: string;
    quantity: string | null;
    unit: string | null;
    unitPrice: string | null;
    lineTotal: string;
}
