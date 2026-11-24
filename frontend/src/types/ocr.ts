export type OCRProvenance = "provider" | "inferred";

export interface OCRScanUsage {
    model: string;
    tokens: { input: number; output: number; thinking: number; cachedInput: number; total: number } | null;
    estimatedCostUsd: number | null;
}

export interface OCRField {
    value: string;
    confidence?: number;
    provenance?: OCRProvenance;
    requiresReview?: boolean;
}

export interface OCRItem {
    description: OCRField;
    quantity: OCRField;
    unitPrice: OCRField;
    lineTotal: OCRField;
}

export interface OCRDraft {
    redemptionNoteConflict?: boolean;
    loyaltyRedemption?: OCRField;
    suggestedExpenseTypeId?: string;
    categoryReason?: string;
    merchant: OCRField;
    date: OCRField;
    currencySuggestion: OCRField;
    subtotal: OCRField;
    tax: OCRField;
    tip: OCRField;
    total: OCRField;
    items: OCRItem[];
}

export interface EditableOCRField extends OCRField {
    edited: boolean;
}

export interface EditableOCRItem {
    id: string;
    description: EditableOCRField;
    quantity: EditableOCRField;
    unit: EditableOCRField;
    unitPrice: EditableOCRField;
    lineTotal: EditableOCRField;
}

export interface EditableOCRDraft {
    loyaltyRedemption?: string;
    merchant: EditableOCRField;
    date: EditableOCRField;
    currencySuggestion: EditableOCRField;
    subtotal: EditableOCRField;
    tax: EditableOCRField;
    tip: EditableOCRField;
    total: EditableOCRField;
    items: EditableOCRItem[];
}

export interface ConfirmedExpenseItem {
    id: string;
    description: string;
    quantity: string;
    unit: string;
    unitPrice: string;
    lineTotal: string;
}

export interface ReviewedReceiptDraft {
    loyaltyRedemption?: string;
    expenseTypeId?: string;
    merchant: string;
    date: string;
    currencySuggestion: string;
    subtotal: string;
    tax: string;
    tip: string;
    total: string;
    items: ConfirmedExpenseItem[];
}
