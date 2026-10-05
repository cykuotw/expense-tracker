import {
    cleanup,
    fireEvent,
    render,
    screen,
    waitFor,
    within,
} from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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

const currencyMetadata = [
    {
        code: "CAD",
        displayName: "Canadian Dollar",
        minorUnitDigits: 2,
        amountDigits: 2,
    },
];

const baseExpense = {
    expenseId: "expense-1",
    description: "Dinner",
    createdByUserID: "user-1",
    createdByUsername: "Current",
    expenseTypeId: "type-1",
    expenseType: "General",
    expenseCategory: "Uncategorized",
    subTotal: "10.00",
    taxFeeTip: "0.00",
    total: "10.00",
    currency: "CAD",
    expenseTime: "2026-01-01T00:00:00Z",
    occurredOn: "2026-01-01",
    invoiceUrl: "",
    currentUser: "user-1",
    groupId: "group-1",
    allocation: { mode: "equal", participants: [] },
    items: [],
    ledgers: [],
};

function renderExpenseDetail() {
    return render(
        <MemoryRouter initialEntries={["/expense/expense-1"]}>
            <Routes>
                <Route path="/expense/:id" element={<ExpenseDetail />} />
                <Route path="/group/:id" element={<div>Group page</div>} />
            </Routes>
        </MemoryRouter>
    );
}

describe("ExpenseDetail", () => {
    afterEach(() => {
        cleanup();
    });

    beforeEach(() => {
        apiFetchMock.mockReset();
        apiFetchMock.mockImplementation(
            (path: string, options?: RequestInit) => {
                if (path === "/currencies") {
                    return Promise.resolve(jsonResponse(currencyMetadata));
                }
                if (
                    path === "/delete_expense/expense-1" &&
                    options?.method === "PUT"
                ) {
                    return Promise.resolve(jsonResponse({}));
                }
                return Promise.resolve(jsonResponse(baseExpense));
            }
        );
    });

    it("retains archived expense history without edit or delete controls", async () => {
        apiFetchMock.mockImplementation((path: string) => Promise.resolve(jsonResponse(path === "/currencies" ? currencyMetadata : { ...baseExpense, groupIsActive: false })));
        renderExpenseDetail();
        expect(await screen.findByText("Archived group · Read-only")).toBeVisible();
        expect(screen.queryByRole("link", { name: /Edit expense/i })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Delete expense/i })).not.toBeInTheDocument();
        expect(screen.getAllByText("10.00 CAD")).toHaveLength(1);
        expect(screen.getAllByRole("link", { name: /Back to group/i })).toHaveLength(2);
    });

    it("renders a valid expense with no ledger entries instead of failing blank", async () => {
        renderExpenseDetail();

        const headings = await screen.findAllByRole("heading", { name: "Dinner" });
        expect(headings).toHaveLength(2);
        expect(headings[0]).toBeVisible();
        expect(screen.getByTestId("expense-category-icon").querySelector("svg")).toBeInTheDocument();
        expect(screen.getByText("No split details are available.")).toBeVisible();
        expect(
            screen.queryByText("No itemized receipt details were saved.")
        ).not.toBeInTheDocument();
        expect(screen.getByText("Jan 1, 2026")).toBeVisible();
        expect(screen.getByText("Current")).toBeVisible();
        expect(screen.getAllByText("10.00 CAD")).toHaveLength(1);
        expect(screen.queryByText(/Added by Current on/i)).not.toBeInTheDocument();
        expect(
            screen.queryByRole("link", { name: "View receipt image" })
        ).not.toBeInTheDocument();

        const editExpense = screen.getAllByRole("link", {
            name: /Edit expense/i,
        });
        const backToGroup = screen.getAllByRole("link", {
            name: /Back to group/i,
        });

        expect(editExpense).toHaveLength(2);
        expect(backToGroup).toHaveLength(2);
        editExpense.forEach((link) => {
            expect(link).toHaveAttribute("href", "/expense/expense-1/edit");
            expect(link.closest(".page-header")).toContainElement(link);
        });
        backToGroup.forEach((link) => {
            expect(link).toHaveAttribute("href", "/group/group-1");
        });
        const desktopBackLink = backToGroup.find((link) =>
            link.classList.contains("desktop-back-link")
        );
        expect(desktopBackLink).toBeDefined();
        expect(desktopBackLink?.closest(".page-header")).toBeNull();
        expect(desktopBackLink?.parentElement).toHaveClass("desktop-page-utility");
        expect(desktopBackLink?.parentElement?.nextElementSibling).toHaveClass("desktop-page-header");
    });

    it("shows every split, ordered receipt item, totals, and the receipt link", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/currencies") {
                return Promise.resolve(jsonResponse(currencyMetadata));
            }
            return Promise.resolve(
                jsonResponse({
                    ...baseExpense,
                    expenseType: "Restaurant",
                    expenseCategory: "Food",
                    subTotal: "25.00",
                    taxFeeTip: "3.40",
                    total: "28.40",
                    invoiceUrl: "https://example.test/receipt.jpg",
                    items: [
                        {
                            itemId: "drink",
                            itemName: "Drink",
                            itemSubTotal: "8.00",
                            description: "Sparkling water",
                            quantity: "2",
                            unit: "bottles",
                            unitPrice: "4.00",
                            lineTotal: "8.00",
                            position: 1,
                        },
                        {
                            itemId: "burger",
                            itemName: "Burger",
                            itemSubTotal: "17.00",
                            description: "Burger",
                            lineTotal: "17.00",
                            position: 0,
                        },
                    ],
                    ledgers: [
                        {
                            id: "ledger-1",
                            lenderUserId: "user-1",
                            lenderUsername: "Current",
                            borrowerUserId: "user-2",
                            borrowerUsername: "Jamie",
                            share: "14.20",
                        },
                        {
                            id: "ledger-2",
                            lenderUserId: "user-3",
                            lenderUsername: "Alex",
                            borrowerUserId: "user-1",
                            borrowerUsername: "Current",
                            share: "3.00",
                        },
                    ],
                })
            );
        });

        renderExpenseDetail();

        expect(
            await screen.findByRole("heading", {
                name: "Paid by You and Alex",
            })
        ).toBeVisible();
        expect(screen.getByText("Jamie owes you")).toBeVisible();
        expect(screen.getByText("You owe Alex")).toBeVisible();
        expect(screen.queryByText("25.00 CAD")).not.toBeInTheDocument();
        expect(screen.queryByText("3.40 CAD")).not.toBeInTheDocument();
        expect(screen.getAllByText("28.40 CAD")).toHaveLength(1);

        const receiptSection = screen.queryByRole("region", {
            name: "Itemized breakdown",
        });
        expect(receiptSection).not.toBeInTheDocument();
        expect(screen.queryByText("Sparkling water")).not.toBeInTheDocument();

        const receiptLink = screen.getByRole("link", {
            name: "View receipt image",
        });
        expect(receiptLink).toHaveAttribute(
            "href",
            "https://example.test/receipt.jpg"
        );
        expect(receiptLink).toHaveAttribute("target", "_blank");
        expect(receiptLink).toHaveAttribute("rel", "noreferrer");
    });

    it("describes soft deletion accurately and submits the existing mutation", async () => {
        renderExpenseDetail();

        const deleteButton = await screen.findByRole("button", {
            name: "Delete expense",
        });
        fireEvent.click(deleteButton);

        const dialog = screen.getByRole("alertdialog");
        expect(
            within(dialog).getByRole("heading", {
                name: "Delete this expense?",
            })
        ).toBeVisible();
        expect(dialog).toHaveTextContent(
            "The retained record is not permanently purged."
        );
        expect(dialog).not.toHaveTextContent(
            "permanently deletes the expense"
        );

        fireEvent.click(
            within(dialog).getByRole("button", { name: "Delete expense" })
        );

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/delete_expense/expense-1",
                expect.objectContaining({ method: "PUT" })
            );
        });
        expect(await screen.findByText("Group page")).toBeVisible();
    });

    it("keeps loading and load-error states accessible", async () => {
        let resolveExpense: ((response: Response) => void) | undefined;
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/currencies") {
                return Promise.resolve(jsonResponse(currencyMetadata));
            }
            return new Promise<Response>((resolve) => {
                resolveExpense = resolve;
            });
        });

        const view = renderExpenseDetail();
        expect(screen.getByRole("status")).toHaveAttribute(
            "aria-busy",
            "true"
        );
        expect(
            screen.getByRole("heading", { name: "Loading expense" })
        ).toBeVisible();

        resolveExpense?.(
            jsonResponse({ message: "Expense could not be found." }, 404)
        );
        expect(
            await screen.findByRole("heading", {
                name: "Unable to load expense",
            })
        ).toBeVisible();
        expect(screen.getByText("Expense could not be found.")).toBeVisible();
        expect(
            screen.getByRole("link", { name: "Back to groups" })
        ).toHaveAttribute("href", "/");
        view.unmount();
    });
});
