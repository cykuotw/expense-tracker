import { API_URL } from "../configs/config";
import { getResponseError, apiFetch } from "./api";
import type {
    EditableOCRDraft,
    EditableOCRField,
    OCRDraft,
    OCRField,
    OCRScanUsage,
} from "../types/ocr";
import type { PreparedReceipt } from "./receiptEditor";
import type { ConfirmedReceiptScan, ExpenseTypeItem } from "../types/expense";

export const OCR_CLIENT_TIMEOUT_MS = 60_000;
const MAX_DRAFT_ITEMS = 200;

interface CapabilityResponse {
    token: string;
    requestId: string;
    contentType: string;
    maxBytes: number;
    expiresAt: string;
}

export class ReceiptOCRError extends Error {
    constructor(
        message: string,
        readonly code: string,
    ) {
        super(message);
        this.name = "ReceiptOCRError";
    }
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseField(value: unknown): OCRField {
    if (!isRecord(value) || typeof value.value !== "string") {
        throw new ReceiptOCRError(
            "The receipt draft was invalid. Continue with manual entry.",
            "invalid_draft",
        );
    }
    const field: OCRField = { value: value.value };
    if (typeof value.confidence === "number" && Number.isFinite(value.confidence)) {
        field.confidence = value.confidence;
    }
    if (value.provenance === "provider" || value.provenance === "inferred") {
        field.provenance = value.provenance;
    }
    if (typeof value.requiresReview === "boolean") {
        field.requiresReview = value.requiresReview;
    }
    return field;
}

export function parseOCRDraft(value: unknown): OCRDraft {
    if (!isRecord(value) || !Array.isArray(value.items) || value.items.length > MAX_DRAFT_ITEMS) {
        throw new ReceiptOCRError(
            "The receipt draft was invalid. Continue with manual entry.",
            "invalid_draft",
        );
    }
    return {
        ...(typeof value.suggestedExpenseTypeId === "string" && value.suggestedExpenseTypeId.length <= 36 &&
            typeof value.categoryReason === "string" && value.categoryReason.trim() && new TextEncoder().encode(value.categoryReason).length <= 500
            ? { suggestedExpenseTypeId: value.suggestedExpenseTypeId, categoryReason: value.categoryReason.trim() } : {}),
        ...(value.redemptionNoteConflict === true ? { redemptionNoteConflict: true } : {}),
        ...(value.loyaltyRedemption === undefined ? {} : { loyaltyRedemption: parseField(value.loyaltyRedemption) }),
        merchant: parseField(value.merchant),
        date: parseField(value.date),
        currencySuggestion: parseField(value.currencySuggestion),
        subtotal: parseField(value.subtotal),
        tax: parseField(value.tax),
        tip: parseField(value.tip),
        total: parseField(value.total),
        items: value.items.map((item) => {
            if (!isRecord(item)) {
                throw new ReceiptOCRError(
                    "The receipt draft was invalid. Continue with manual entry.",
                    "invalid_draft",
                );
            }
            return {
                description: parseField(item.description),
                quantity: parseField(item.quantity),
                unitPrice: parseField(item.unitPrice),
                lineTotal: parseField(item.lineTotal),
            };
        }),
    };
}

function parseCapability(value: unknown): CapabilityResponse {
    if (!isRecord(value)) {
        throw new ReceiptOCRError("Receipt scanning is unavailable.", "invalid_capability");
    }
    const token = value.token;
    const requestId = value.requestId;
    const contentType = value.contentType;
    const maxBytes = value.maxBytes;
    const expiresAt = value.expiresAt;
    if (
        typeof token !== "string" || !token ||
        typeof requestId !== "string" || !requestId ||
        typeof contentType !== "string" || !contentType ||
        typeof maxBytes !== "number" || !Number.isFinite(maxBytes) || maxBytes <= 0 ||
        typeof expiresAt !== "string" || !expiresAt
    ) {
        throw new ReceiptOCRError("Receipt scanning is unavailable.", "invalid_capability");
    }
    return { token, requestId, contentType, maxBytes, expiresAt };
}

async function errorFromResponse(response: Response, fallback: string) {
    const error = await getResponseError(response, fallback);
    if (error.code === "receipt_ocr_not_granted") {
        return new ReceiptOCRError(
            "Receipt scanning is no longer enabled for this account. Continue with manual entry.",
            error.code,
        );
    }
    if (error.code === "receipt_storage_unavailable") {
        return new ReceiptOCRError(
            "Keeping a receipt photo is unavailable right now. Turn off Keep receipt to scan values only.",
            error.code,
        );
    }
    if (response.status === 429) {
        return new ReceiptOCRError(
            "Receipt scanning is busy. Wait a moment, retry, or continue with manual entry.",
            error.code ?? "ocr_throttled",
        );
    }
    return new ReceiptOCRError(error.message || fallback, error.code ?? "ocr_failed");
}

export interface ReceiptDraftResult {
    scan: ConfirmedReceiptScan;
	usage?: OCRScanUsage | null;
    draft: OCRDraft;
    receiptToken: string | null;
    receiptExpiresAt: string | null;
}

// Optional telemetry must never make a usable receipt draft fail.
export function parseOCRScanUsage(value: unknown): OCRScanUsage | null {
    if (!isRecord(value) || typeof value.model !== "string" || !/^gemini-[a-zA-Z0-9.-]{1,80}$/.test(value.model)) return null;
    const raw = value.tokens;
    let tokens: OCRScanUsage["tokens"] = null;
    if (isRecord(raw)) {
        const { input, output, thinking, cachedInput, total } = raw;
        const count = (n: unknown): n is number => typeof n === "number" && Number.isSafeInteger(n) && n >= 0 && n <= 1_000_000_000;
        if (count(input) && count(output) && count(thinking) && count(cachedInput) && count(total)
            && cachedInput <= input && input + output + thinking <= total) {
            tokens = { input, output, thinking, cachedInput, total };
        }
    }
    const cost = value.estimatedCostUsd;
    return {
        model: value.model,
        tokens,
        estimatedCostUsd: tokens && typeof cost === "number" && Number.isFinite(cost) && cost >= 0 ? cost : null,
    };
}

// Multipart normalizes line endings. Go's TrimSpace also trims U+0085.
export function normalizeScanNote(note: string): string {
    return note.replace(/\r\n?/g, "\n").replace(/^[\s\u0085]+|[\s\u0085]+$/gu, "");
}

export async function requestReceiptDraftWithRetention(
    receipt: PreparedReceipt,
    accountID: string,
    keepReceipt: boolean,
    signal?: AbortSignal,
    expenseTypes: ExpenseTypeItem[] = [],
    note = "",
): Promise<ReceiptDraftResult> {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort("timeout"), OCR_CLIENT_TIMEOUT_MS);
    const abort = () => controller.abort(signal?.reason ?? "cancelled");
    signal?.addEventListener("abort", abort, { once: true });

    try {
        const requestId = crypto.randomUUID();
        const contentType = receipt.blob.type || "image/jpeg";
        const capabilityResponse = await apiFetch("/ocr/capabilities", {
            method: "POST",
            signal: controller.signal,
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ requestId, contentType, keepReceipt }),
        });
        if (!capabilityResponse.ok) {
            throw await errorFromResponse(capabilityResponse, "Receipt scanning is unavailable.");
        }
        const capability = parseCapability(await capabilityResponse.json());
        if (
            capability.requestId !== requestId ||
            capability.contentType !== contentType ||
            receipt.blob.size > capability.maxBytes
        ) {
            throw new ReceiptOCRError(
                "The prepared receipt does not match the upload permission. Try preparing it again.",
                "capability_mismatch",
            );
        }

        const body = new FormData();
        body.append("image", receipt.blob, "receipt");
        const categories = expenseTypes.map(({ id, name, category }) => ({ id, name, category }));
        const encodedCategories = JSON.stringify(categories);
        const encoder = new TextEncoder();
        const validCategories = categories.length <= 200 && new Set(categories.map(({ id }) => id)).size === categories.length &&
            categories.every(({ id, name, category }) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id) &&
                name.trim() && category.trim() && encoder.encode(name).length <= 256 && encoder.encode(category).length <= 256) &&
            encoder.encode(encodedCategories).length <= 32 * 1024;
        body.append("categories", validCategories ? encodedCategories : "[]");
        const submittedNote = normalizeScanNote(note);
        if (submittedNote) body.append("note", submittedNote);
        const response = await fetch(`${API_URL}/ocr/drafts`, {
            method: "POST",
            credentials: "omit",
            signal: controller.signal,
            headers: {
                Authorization: `Bearer ${capability.token}`,
                "X-OCR-Account-ID": accountID,
                "X-OCR-Request-ID": requestId,
            },
            body,
        });
        if (!response.ok) {
            throw await errorFromResponse(
                response,
                "The receipt could not be scanned. Retry or continue with manual entry.",
            );
        }
        const value: unknown = await response.json();
        if (!isRecord(value) || value.requestId !== requestId) {
            throw new ReceiptOCRError(
                "The receipt draft was invalid. Continue with manual entry.",
                "invalid_draft",
            );
        }
        const draft = parseOCRDraft(value.draft);
        const receiptToken = typeof value.receiptToken === "string" && value.receiptToken
            ? value.receiptToken : null;
        const receiptExpiresAt = typeof value.receiptExpiresAt === "string"
            ? value.receiptExpiresAt : null;
        if (keepReceipt && (!receiptToken || !receiptExpiresAt)) {
            throw new ReceiptOCRError(
                "The receipt values were read, but the photo could not be saved. Scan again without keeping the photo or retry later.",
                "receipt_storage_unavailable",
            );
        }
        const scan = value.scan;
        if (!isRecord(scan) || scan.requestId !== requestId ||
            typeof scan.proof !== "string" || !scan.proof.trim() ||
            (scan.note === undefined ? "" : scan.note) !== submittedNote) {
            throw new ReceiptOCRError("The scan could not be verified. Scan again or continue with manual entry.", "invalid_draft");
        }
        return {
            draft, receiptToken, receiptExpiresAt, usage: parseOCRScanUsage(value.usage),
            scan: { requestId, note: submittedNote, proof: scan.proof },
        };
    } catch (error) {
        if (error instanceof ReceiptOCRError) throw error;
        if (controller.signal.aborted) {
            const timedOut = controller.signal.reason === "timeout";
            throw new ReceiptOCRError(
                timedOut
                    ? "Receipt scanning took too long. Retry or continue with manual entry."
                    : "Receipt scanning was cancelled.",
                timedOut ? "ocr_timeout" : "ocr_cancelled",
            );
        }
        throw new ReceiptOCRError(
            "The receipt could not be scanned. Check your connection, retry, or continue with manual entry.",
            "ocr_network_error",
        );
    } finally {
        window.clearTimeout(timeout);
        signal?.removeEventListener("abort", abort);
    }
}

const editableField = (field: OCRField): EditableOCRField => ({ ...field, edited: false });

export function editableOCRDraft(draft: OCRDraft): EditableOCRDraft {
    return {
        loyaltyRedemption: draft.loyaltyRedemption?.value,
        merchant: editableField(draft.merchant),
        date: editableField(draft.date),
        currencySuggestion: editableField(draft.currencySuggestion),
        subtotal: editableField(draft.subtotal),
        tax: editableField(draft.tax),
        tip: editableField(draft.tip),
        total: editableField(draft.total),
        items: draft.items.map((item) => ({
            id: crypto.randomUUID(),
            description: editableField(item.description),
            quantity: editableField(item.quantity),
            unit: { value: "", edited: false },
            unitPrice: editableField(item.unitPrice),
            lineTotal: editableField(item.lineTotal),
        })),
    };
}


export async function requestReceiptDraft(
    receipt: PreparedReceipt,
    accountID: string,
    signal?: AbortSignal,
): Promise<OCRDraft> {
    return (await requestReceiptDraftWithRetention(receipt, accountID, false, signal)).draft;
}
