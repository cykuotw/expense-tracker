import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ExpenseDetailProvider } from "./ExpenseDetailContext";
import { useExpenseDetail } from "../hooks/ExpenseDetailContextHooks";

const { apiFetchMock, toastErrorMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    toastErrorMock: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
    useParams: () => ({ id: "expense-1" }),
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
    toast: { error: toastErrorMock },
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

const expenseDetail = {
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
    occurredOn: "2026-01-01",
    invoiceUrl: "",
    currentUser: "user-1",
    groupId: "group-1",
    splitRule: "Equally",
    items: [],
    ledgers: [],
};

function ExpenseDetailHarness() {
    const context = useExpenseDetail();

    if (!context.expenseDetail) {
        return <output data-testid="load-error">{context.errorMessage ?? ""}</output>;
    }

    return (
        <form aria-label="delete form" onSubmit={context.handleDeleteExpense}>
            <button type="submit">Delete</button>
        </form>
    );
}

describe("ExpenseDetailProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1") {
                    return Promise.resolve(jsonResponse(expenseDetail));
                }
                if (
                    path === "/delete_expense/expense-1" &&
                    init.method === "PUT"
                ) {
                    return Promise.resolve(
                        jsonResponse({ error: "delete not permitted" }, 403)
                    );
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );
    });

    afterEach(() => {
        cleanup();
    });

    it("shows the delete error without redirecting", async () => {
        const initialHref = window.location.href;
        render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );
        await screen.findByRole("form", { name: "delete form" });

        fireEvent.submit(screen.getByRole("form", { name: "delete form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "delete not permitted"
            );
        });
        expect(window.location.href).toBe(initialHref);
    });

    it("exposes an expense-load failure for the page to render", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense/expense-1") {
                return Promise.resolve(jsonResponse({ error: "expense unavailable" }, 404));
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("load-error")).toHaveTextContent(
                "expense unavailable"
            );
        });
    });
});
