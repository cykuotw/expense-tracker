import { API_URL } from "../configs/config";
import { getResponseError, apiFetch } from "./api";
import type {
    EditableOCRDraft,
    EditableOCRField,
    OCRDraft,
    OCRField,
} from "../types/ocr";
import type { PreparedReceipt } from "./receiptEditor";

export const OCR_CLIENT_TIMEOUT_MS = 30_000;
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
    if (response.status === 429) {
        return new ReceiptOCRError(
            "Receipt scanning is busy. Wait a moment, retry, or continue with manual entry.",
            error.code ?? "ocr_throttled",
        );
    }
    return new ReceiptOCRError(error.message || fallback, error.code ?? "ocr_failed");
}

export async function requestReceiptDraft(
    receipt: PreparedReceipt,
    accountID: string,
    signal?: AbortSignal,
): Promise<OCRDraft> {
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
            body: JSON.stringify({ requestId, contentType }),
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

        const response = await fetch(`${API_URL}/ocr/drafts`, {
            method: "POST",
            credentials: "omit",
            signal: controller.signal,
            headers: {
                Authorization: `Bearer ${capability.token}`,
                "Content-Type": contentType,
                "X-OCR-Account-ID": accountID,
                "X-OCR-Request-ID": requestId,
            },
            body: receipt.blob,
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
        return parseOCRDraft(value.draft);
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

