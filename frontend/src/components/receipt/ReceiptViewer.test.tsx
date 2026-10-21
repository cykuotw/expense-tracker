import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import ReceiptViewer from "./ReceiptViewer";

const { api } = vi.hoisted(() => ({ api: vi.fn() }));
vi.mock("@/lib/api", () => ({ apiFetch: api, getResponseErrorMessage: async () => "Receipt unavailable" }));
afterEach(() => { cleanup(); api.mockReset(); vi.useRealTimers(); });

describe("ReceiptViewer", () => {
    it("cancels the old expiry timer before refreshing a receipt view", async () => {
        vi.useFakeTimers();
        let finish!: (value: unknown) => void;
        api.mockResolvedValueOnce({ ok: true, json: async () => ({ url: "https://receipts.example/old", expiresAt: new Date(Date.now() + 1000).toISOString() }) }).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
        render(<ReceiptViewer expenseId="first" />);
        await act(async () => { fireEvent.click(screen.getByRole("button", { name: "View receipt" })); });
        await act(async () => { vi.advanceTimersByTime(500); fireEvent.click(screen.getByRole("button", { name: "Refresh receipt view" })); });
        act(() => { vi.advanceTimersByTime(600); });
        await act(async () => { finish({ ok: true, json: async () => ({ url: "https://receipts.example/new", expiresAt: new Date(Date.now() + 45_000).toISOString() }) }); });
        expect(screen.getByAltText("Saved receipt")).toHaveAttribute("src", "https://receipts.example/new");
        expect(screen.queryByText(/view expired/)).toBeNull();
    });
    it("loads only on request, hides on expense change, and never reuses the old view", async () => {
        api.mockResolvedValue({ ok: true, json: async () => ({ url: "https://receipts.example/first?signature=private", expiresAt: new Date(Date.now() + 45_000).toISOString() }) });
        const view = render(<ReceiptViewer expenseId="first" />);
        expect(api).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "View receipt" }));
        expect(await screen.findByAltText("Saved receipt")).toHaveAttribute("referrerpolicy", "no-referrer");
        view.rerender(<ReceiptViewer expenseId="second" />);
        expect(screen.queryByAltText("Saved receipt")).toBeNull();
        expect(api).toHaveBeenCalledTimes(1);
        fireEvent.click(screen.getByRole("button", { name: "View receipt" }));
        await waitFor(() => expect(api).toHaveBeenLastCalledWith("/expense/second/receipt/view", expect.anything()));
    });

    it("offers a fresh request after a failed image load", async () => {
        api.mockResolvedValue({ ok: true, json: async () => ({ url: "https://receipts.example/photo", expiresAt: new Date(Date.now() + 45_000).toISOString() }) });
        render(<ReceiptViewer expenseId="first" />);
        fireEvent.click(screen.getByRole("button", { name: "View receipt" }));
        fireEvent.error(await screen.findByAltText("Saved receipt"));
        expect(screen.queryByAltText("Saved receipt")).toBeNull();
        expect(screen.getByRole("status")).toHaveTextContent("Open it again to retry");
        fireEvent.click(screen.getByRole("button", { name: "View receipt" }));
        await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
    });
});
