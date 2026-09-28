import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("./api", async () => {
    const actual = await vi.importActual<typeof import("./api")>("./api");
    return { ...actual, apiFetch: (...args: unknown[]) => apiFetchMock(...args) };
});
vi.mock("../configs/config", () => ({ API_URL: "https://api.example.test/api/v0" }));

import { OCR_CLIENT_TIMEOUT_MS, parseOCRDraft, ReceiptOCRError, requestReceiptDraft } from "./receiptOcr";

const field = (value: string, provenance = "provider") => ({ value, provenance });
const draft = {
    merchant: field("Cafe"),
    date: { ...field("2026-09-27"), requiresReview: true },
    currencySuggestion: field("CAD"),
    subtotal: field("5.00"),
    tax: field("0.65"),
    tip: field(""),
    total: field("5.65"),
    items: [{
        description: field("Coffee"),
        quantity: field("1", "inferred"),
        unitPrice: field("5.00"),
        lineTotal: field("5.00"),
    }],
};
const receipt = { blob: new Blob(["jpeg"], { type: "image/jpeg" }), width: 10, height: 20, quality: 0.8 };

function jsonResponse(value: unknown, status = 200) {
    return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}

describe("receipt OCR client", () => {
    beforeEach(() => {
        vi.spyOn(crypto, "randomUUID").mockReturnValue("00000000-0000-4000-8000-000000000001");
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    it("mints a capability then uploads bytes without browser credentials", async () => {
        apiFetchMock.mockResolvedValue(jsonResponse({
            token: "token",
            requestId: "00000000-0000-4000-8000-000000000001",
            contentType: "image/jpeg",
            maxBytes: 3_670_016,
            expiresAt: "2026-09-27T12:00:00Z",
        }, 201));
        const fetchMock = vi.fn().mockResolvedValue(jsonResponse({
            requestId: "00000000-0000-4000-8000-000000000001",
            draft,
        }));
        vi.stubGlobal("fetch", fetchMock);

        await expect(requestReceiptDraft(receipt, "account-1")).resolves.toMatchObject({ merchant: { value: "Cafe" } });
        expect(apiFetchMock).toHaveBeenCalledWith("/ocr/capabilities", expect.objectContaining({ method: "POST" }));
        expect(fetchMock).toHaveBeenCalledWith(
            "https://api.example.test/api/v0/ocr/drafts",
            expect.objectContaining({
                credentials: "omit",
                body: receipt.blob,
                headers: expect.objectContaining({
                    Authorization: "Bearer token",
                    "X-OCR-Account-ID": "account-1",
                    "X-OCR-Request-ID": "00000000-0000-4000-8000-000000000001",
                }),
            }),
        );
    });

    it("maps a revoked grant to an actionable safe error", async () => {
        apiFetchMock.mockResolvedValue(jsonResponse({
            error: "not available",
            code: "receipt_ocr_not_granted",
        }, 403));
        await expect(requestReceiptDraft(receipt, "account-1")).rejects.toMatchObject({
            code: "receipt_ocr_not_granted",
        });
    });

    it("rejects malformed provider-neutral drafts", () => {
        expect(() => parseOCRDraft({ items: [{ description: null }] })).toThrow(ReceiptOCRError);
    });

    it("aborts capability acquisition at the bounded client timeout", async () => {
        expect(OCR_CLIENT_TIMEOUT_MS).toBe(30_000);
        vi.useFakeTimers();
        apiFetchMock.mockImplementation((_path: string, init: RequestInit) => new Promise((_resolve, reject) => {
            init.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true });
        }));
        const pending = requestReceiptDraft(receipt, "account-1");
        const rejection = expect(pending).rejects.toMatchObject({ code: "ocr_timeout" });
        await vi.advanceTimersByTimeAsync(OCR_CLIENT_TIMEOUT_MS);
        await rejection;
    });
});
