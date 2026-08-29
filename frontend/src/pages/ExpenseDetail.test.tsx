import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import ExpenseDetail from "./ExpenseDetail";

const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
}));

vi.mock("../lib/api", async () => {
    const actual = await vi.importActual<typeof import("../lib/api")>(
        "../lib/api"
    );
    return {
        ...actual,
        apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    };
});

vi.mock("react-hot-toast", () => ({
    toast: { error: vi.fn() },
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

describe("ExpenseDetail", () => {
    it("renders a valid expense with no ledger entries instead of failing blank", async () => {
        apiFetchMock.mockResolvedValue(
            jsonResponse({
                expenseId: "expense-1",
                description: "Dinner",
                createdByUserID: "user-1",
                createdByUsername: "Current",
                expenseTypeId: "type-1",
                expenseType: "General",
                subTotal: "10.00",
                taxFeeTip: "0.00",
                total: "10.00",
                currency: "CAD",
                expenseTime: "2026-01-01T00:00:00Z",
                invoiceUrl: "",
                currentUser: "user-1",
                groupId: "group-1",
                splitRule: "Equally",
                items: [],
                ledgers: [],
            })
        );

        render(
            <MemoryRouter initialEntries={["/expense/expense-1"]}>
                <Routes>
                    <Route path="/expense/:id" element={<ExpenseDetail />} />
                </Routes>
            </MemoryRouter>
        );

        expect(await screen.findByRole("heading", { name: "Dinner" })).toBeVisible();
        expect(screen.getByText("No split details are available.")).toBeVisible();
    });
});
