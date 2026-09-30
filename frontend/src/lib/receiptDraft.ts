import { decimalToUnits, unitsToDecimal } from "./money";
import type { ConfirmedExpenseItem } from "../types/ocr";
import type { ItemCreateData } from "../types/item";

export interface ReceiptExpenseValues {
    merchant: string;
    subtotal: string;
    tax: string;
    tip: string;
    total: string;
    items: ConfirmedExpenseItem[];
}

export interface ReceiptExpenseValidation {
    valid: boolean;
    message: string | null;
    section?: "amounts" | "items";
    subTotal?: string;
    taxFeeTip?: string;
    items?: ItemCreateData[];
}

const optionalUnits = (value: string, digits: number) =>
    value.trim() === "" ? 0n : decimalToUnits(value, digits);

const validQuantity = (value: string) =>
    /^(?:0|[1-9]\d{0,8})(?:\.\d{1,6})?$/.test(value) && Number(value) > 0;

const validUnitPrice = (value: string) =>
    /^(?:0|[1-9]\d{0,6})(?:\.\d{1,3})?$/.test(value);

export function validateReceiptExpense(
    values: ReceiptExpenseValues,
    amountDigits: number | null,
): ReceiptExpenseValidation {
    if (amountDigits === null) {
        return { valid: false, message: "Choose a group with an available currency first.", section: "amounts" };
    }

    const hasBreakdown = [values.subtotal, values.tax, values.tip].some(
        (value) => value.trim() !== "",
    ) || values.items.length > 0;
    if (!hasBreakdown) return { valid: true, message: null, items: [] };

    const totalUnits = decimalToUnits(values.total, amountDigits);
    const subtotalUnits = decimalToUnits(values.subtotal, amountDigits);
    const taxUnits = optionalUnits(values.tax, amountDigits);
    const tipUnits = optionalUnits(values.tip, amountDigits);
    if (
        totalUnits === null || totalUnits <= 0n || subtotalUnits === null ||
        subtotalUnits < 0n || taxUnits === null || taxUnits < 0n ||
        tipUnits === null || tipUnits < 0n
    ) {
        return {
            valid: false,
            message: `Use valid amounts with at most ${amountDigits} decimal places.`,
            section: "amounts",
        };
    }
    if (subtotalUnits + taxUnits + tipUnits !== totalUnits) {
        return { valid: false, message: "Subtotal, tax, and tip must add up to the total.", section: "amounts" };
    }

    let itemUnits = 0n;
    const items: ItemCreateData[] = [];
    for (const item of values.items) {
        const description = item.description.trim();
        const lineUnits = decimalToUnits(item.lineTotal, amountDigits);
        if (!description || Array.from(description).length > 256 || lineUnits === null || lineUnits < 0n) {
            return { valid: false, message: "Each item needs a description and a valid line total.", section: "items" };
        }
        const quantity = item.quantity.trim();
        const unit = item.unit.trim();
        const unitPrice = item.unitPrice.trim();
        if (quantity && !validQuantity(quantity)) {
            return { valid: false, message: "Item quantity must be positive with at most 6 decimal places.", section: "items" };
        }
        if (unit && Array.from(unit).length > 32) {
            return { valid: false, message: "Item unit must be 32 characters or fewer.", section: "items" };
        }
        if (unitPrice && !validUnitPrice(unitPrice)) {
            return { valid: false, message: "Item unit price must be non-negative with at most 3 decimal places.", section: "items" };
        }
        itemUnits += lineUnits;
        items.push({
            description,
            quantity: quantity || null,
            unit: unit || null,
            unitPrice: unitPrice || null,
            lineTotal: unitsToDecimal(lineUnits, amountDigits),
        });
    }
    if (items.length > 0 && itemUnits !== subtotalUnits) {
        return { valid: false, message: "Item line totals must add up to the subtotal.", section: "items" };
    }

    return {
        valid: true,
        message: null,
        subTotal: unitsToDecimal(subtotalUnits, amountDigits),
        taxFeeTip: unitsToDecimal(taxUnits + tipUnits, amountDigits),
        items,
    };
}
