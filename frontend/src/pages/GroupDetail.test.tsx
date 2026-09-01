import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import GroupDetail from "./GroupDetail";

const { groupDetailMock } = vi.hoisted(() => ({
    groupDetailMock: vi.fn(),
}));

vi.mock("../hooks/GroupDetailContextHooks", async (importOriginal) => {
    const actual = await importOriginal<
        typeof import("../hooks/GroupDetailContextHooks")
    >();
    return {
        ...actual,
        useGroupDetail: () => groupDetailMock(),
    };
});

vi.mock("../components/expense/ExpenseCard", () => ({
    default: () => <div>Expense card</div>,
}));

function renderGroupDetail() {
    return render(
        <MemoryRouter>
            <GroupDetail />
        </MemoryRouter>
    );
}

afterEach(() => {
    cleanup();
    groupDetailMock.mockReset();
});

describe("GroupDetail mobile balance summary", () => {
    it("prioritizes mobile expense creation and links member management to edit group", () => {
        groupDetailMock.mockReturnValue({
            groupinfo: {
                groupName: "Trip",
                description: "",
                currency: "CAD",
                groupType: "trip",
                members: [
                    { userId: "user-1", username: "Alex" },
                    { userId: "user-2", username: "Blair" },
                ],
            },
            balance: null,
            unsettledExpenses: [],
            unsettledLoading: false,
            unsettledHasMore: false,
            expenseOrder: "newest",
            expenseListRefreshVersion: 0,
            setExpenseOrder: vi.fn(),
            settledExpenses: [],
            settledLoading: false,
            settledHasMore: false,
            loading: false,
            groupId: "group-1",
            handleSettle: vi.fn(),
            loadMoreUnsettledExpenses: vi.fn(),
            loadSettledExpenses: vi.fn(),
            loadMoreSettledExpenses: vi.fn(),
        });

        renderGroupDetail();

        expect(screen.getByLabelText("Edit group")).toHaveAttribute(
            "href",
            "/group/group-1/edit"
        );
        expect(screen.getByRole("link", { name: "Manage 2 members" })).toHaveAttribute(
            "href",
            "/group/group-1/edit#members"
        );
        expect(screen.getByRole("link", { name: "Add expense" })).toHaveClass(
            "group-add-expense-fab"
        );
        expect(screen.getByRole("link", { name: "Add expense" })).toHaveAttribute(
            "href",
            "/create_expense?g=group-1"
        );
    });

    it("shows two balances initially and expands the compact mobile summary", () => {
        const setExpenseOrder = vi.fn();
        groupDetailMock.mockReturnValue({
            groupinfo: { groupName: "Trip", description: "", currency: "CAD" },
            balance: {
                currency: "CAD",
                currentUser: "current-user",
                balances: [
                    {
                        id: "balance-1",
                        senderUserId: "alex",
                        senderUsername: "Alex",
                        receiverUserId: "current-user",
                        receiverUsername: "Current",
                        balance: "10.00",
                    },
                    {
                        id: "balance-2",
                        senderUserId: "current-user",
                        senderUsername: "Current",
                        receiverUserId: "blair",
                        receiverUsername: "Blair",
                        balance: "20.00",
                    },
                    {
                        id: "balance-3",
                        senderUserId: "casey",
                        senderUsername: "Casey",
                        receiverUserId: "current-user",
                        receiverUsername: "Current",
                        balance: "30.00",
                    },
                ],
            },
            unsettledExpenses: [],
            unsettledLoading: false,
            unsettledHasMore: false,
            expenseOrder: "newest",
            setExpenseOrder,
            settledExpenses: [],
            settledLoading: false,
            settledHasMore: false,
            loading: false,
            groupId: "group-1",
            handleSettle: vi.fn(),
            loadMoreUnsettledExpenses: vi.fn(),
            loadSettledExpenses: vi.fn(),
            loadMoreSettledExpenses: vi.fn(),
        });

        renderGroupDetail();

        const summary = screen.getByTestId("mobile-balance-summary");
        expect(within(summary).getByText("Alex owes you")).toBeVisible();
        expect(within(summary).getByText("You owe Blair")).toBeVisible();
        expect(within(summary).queryByText("Casey owes you")).toBeNull();

        fireEvent.click(screen.getByRole("button", { name: "View all 3 balances" }));

        expect(within(summary).getByText("Casey owes you")).toBeVisible();
        expect(
            screen.getByRole("button", { name: "Show fewer balances" })
        ).toHaveAttribute("aria-expanded", "true");

        const expenseOrder = screen.getByLabelText("Expense order");
        expect(expenseOrder).toHaveValue("newest");
        fireEvent.change(expenseOrder, { target: { value: "oldest" } });
        expect(setExpenseOrder).toHaveBeenCalledWith("oldest");
    });

    it("uses full-width mobile load controls and compact desktop controls", () => {
        const loadMoreUnsettledExpenses = vi.fn();
        const loadMoreSettledExpenses = vi.fn();
        groupDetailMock.mockReturnValue({
            groupinfo: { groupName: "Trip", description: "", currency: "CAD" },
            balance: null,
            unsettledExpenses: [{ expenseId: "unsettled-1" }],
            unsettledLoading: false,
            unsettledHasMore: true,
            expenseOrder: "newest",
            setExpenseOrder: vi.fn(),
            settledExpenses: [{ expenseId: "settled-1" }],
            settledLoading: false,
            settledHasMore: true,
            loading: false,
            groupId: "group-1",
            handleSettle: vi.fn(),
            loadMoreUnsettledExpenses,
            loadSettledExpenses: vi.fn(),
            loadMoreSettledExpenses,
        });

        renderGroupDetail();

        const unsettledButton = screen.getByRole("button", {
            name: "Load more unsettled expenses",
        });
        expect(unsettledButton).toHaveClass("w-full", "sm:w-auto");
        fireEvent.click(unsettledButton);
        expect(loadMoreUnsettledExpenses).toHaveBeenCalledOnce();

        fireEvent.click(screen.getByRole("button", { name: "Load Settled Expenses" }));
        const settledButton = screen.getByRole("button", {
            name: "Load more settled expenses",
        });
        expect(settledButton).toHaveClass("w-full", "sm:w-auto");
        fireEvent.click(settledButton);
        expect(loadMoreSettledExpenses).toHaveBeenCalledOnce();
    });

    it("shows terminal copy instead of load controls after the last page", () => {
        groupDetailMock.mockReturnValue({
            groupinfo: { groupName: "Trip", description: "", currency: "CAD" },
            balance: null,
            unsettledExpenses: [{ expenseId: "unsettled-1" }],
            unsettledLoading: false,
            unsettledHasMore: false,
            expenseOrder: "newest",
            setExpenseOrder: vi.fn(),
            settledExpenses: [{ expenseId: "settled-1" }],
            settledLoading: false,
            settledHasMore: false,
            loading: false,
            groupId: "group-1",
            handleSettle: vi.fn(),
            loadMoreUnsettledExpenses: vi.fn(),
            loadSettledExpenses: vi.fn(),
            loadMoreSettledExpenses: vi.fn(),
        });

        renderGroupDetail();

        expect(screen.getByText("No more unsettled expenses")).toBeVisible();
        expect(
            screen.queryByRole("button", { name: "Load more unsettled expenses" })
        ).toBeNull();

        fireEvent.click(screen.getByRole("button", { name: "Load Settled Expenses" }));
        expect(screen.getByText("No more settled expenses")).toBeVisible();
        expect(
            screen.queryByRole("button", { name: "Load more settled expenses" })
        ).toBeNull();
    });
});
