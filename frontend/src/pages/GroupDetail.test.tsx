import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
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

describe("GroupDetail mobile balance summary", () => {
    it("shows two balances initially and expands the compact mobile summary", () => {
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
    });
});
