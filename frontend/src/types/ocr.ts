export type OCRProvenance = "provider" | "inferred";

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
    merchant: string;
    date: string;
    currencySuggestion: string;
    subtotal: string;
    tax: string;
    tip: string;
    total: string;
    items: ConfirmedExpenseItem[];
}
