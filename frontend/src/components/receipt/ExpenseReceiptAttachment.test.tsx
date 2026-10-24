import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ExpenseReceiptAttachment from "./ExpenseReceiptAttachment";

const { api, scan } = vi.hoisted(() => ({ api: vi.fn(), scan: vi.fn() }));
vi.mock("@/lib/api", () => ({ apiFetch: api, getResponseErrorMessage: vi.fn() }));
vi.mock("@/lib/receiptOcr", () => ({ requestReceiptDraftWithRetention: scan }));
vi.mock("@/hooks/AuthContextHooks", () => ({ useAuth: () => ({ userID: "user-1" }) }));

afterEach(cleanup);
beforeEach(() => {
    vi.clearAllMocks();
    api.mockResolvedValue({ ok: true, json: async () => ({ capabilities: { receiptRetention: true } }) });
});

describe("ExpenseReceiptAttachment", () => {
    it.each([undefined, { status: "finalized", contentType: "image/jpeg", byteSize: 1, width: 1, height: 1 }])("offers existing-photo upload for a new or replacement receipt", async (receipt) => {
        render(<ExpenseReceiptAttachment expenseId="expense-1" initialReceipt={receipt} />);
        fireEvent.click(await screen.findByRole("button", { name: receipt ? "Replace photo" : "Upload photo" }));
        const library = await screen.findByLabelText("Choose photo");
        expect(library).toHaveAttribute("type", "file");
        expect(library).toHaveAttribute("accept", "image/*");
        expect(library).not.toHaveAttribute("capture");
        expect(screen.getByLabelText("Take photo")).toHaveAttribute("capture", "environment");
        expect(scan).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "Continue with manual entry" }));
        expect(screen.queryByLabelText("Choose photo")).not.toBeInTheDocument();
        expect(api).toHaveBeenCalledTimes(1);
    });
});
