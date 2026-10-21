import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, Link } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import ItemHistory from "./ItemHistory";

const { api } = vi.hoisted(() => ({ api: vi.fn() }));
vi.mock("@/lib/api", () => ({ apiFetch: api, getResponseErrorMessage: async () => "History unavailable" }));
afterEach(() => { cleanup(); api.mockReset(); });
function view() { return render(<MemoryRouter initialEntries={["/group/first/item-history"]}><Link to="/group/second/item-history">Other group</Link><Routes><Route path="/group/:id/item-history" element={<ItemHistory />} /></Routes></MemoryRouter>); }

describe("ItemHistory", () => {
    it("separates currencies and loads subsequent pages for the submitted search", async () => {
        const row = { itemId: "1", expenseId: "expense", description: "Milk", merchant: "Shop", date: "2022-03-18", currency: "CAD", quantity: null, unitPrice: null, lineTotal: "5.39" };
        api.mockResolvedValueOnce({ ok: true, json: async () => ({ items: [row], nextCursor: "next" }) }).mockResolvedValueOnce({ ok: true, json: async () => ({ items: [{ ...row, itemId: "2", currency: "USD" }] }) });
        view();
        fireEvent.change(screen.getByLabelText("Item description"), { target: { value: "Milk" } });
        fireEvent.click(screen.getByRole("button", { name: "Search history" }));
        expect(await screen.findByRole("region", { name: "CAD purchases" })).toBeInTheDocument();
        fireEvent.change(screen.getByLabelText("Item description"), { target: { value: "Other" } });
        fireEvent.click(screen.getByRole("button", { name: "Load more purchases" }));
        expect(await screen.findByRole("region", { name: "USD purchases" })).toBeInTheDocument();
        expect(api).toHaveBeenLastCalledWith("/group/first/item-history?q=Milk&cursor=next", expect.anything());
        expect(screen.queryByText(/^Unit price:/)).toBeNull();
    });

    it("allows searching a new group after aborting an in-flight search", async () => {
        api.mockImplementation(() => new Promise(() => {}));
        view();
        fireEvent.change(screen.getByLabelText("Item description"), { target: { value: "Milk" } });
        fireEvent.click(screen.getByRole("button", { name: "Search history" }));
        fireEvent.click(screen.getByRole("link", { name: "Other group" }));
        fireEvent.change(screen.getByLabelText("Item description"), { target: { value: "Milk" } });
        await waitFor(() => expect(screen.getByRole("button", { name: "Search history" })).not.toBeDisabled());
        expect(screen.queryByText("Searching…")).toBeNull();
    });
});
