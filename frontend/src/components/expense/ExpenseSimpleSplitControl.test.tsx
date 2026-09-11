import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { calculateExpenseAllocation } from "../../lib/expenseAllocation";
import { ExpenseAllocation } from "../../types/allocation";
import { GroupMember } from "../../types/group";
import ExpenseSimpleSplitControl from "./ExpenseSimpleSplitControl";

const members: GroupMember[] = [
    { userId: "other", username: "Alice" },
    { userId: "current", username: "You" },
];
const threeMembers: GroupMember[] = [
    { userId: "alice", username: "Alice" },
    { userId: "bob", username: "Bob" },
    { userId: "current", username: "You" },
];
const equalAllocation: ExpenseAllocation = {
    mode: "equal",
    participants: members.map(({ userId }) => ({ userId })),
};

function renderControl({
    allocation = equalAllocation,
    payerUserId = "current",
    onAllocationChange = vi.fn<(allocation: ExpenseAllocation) => void>(),
    onPayerChange = vi.fn<(payerUserId: string) => void>(),
}: {
    allocation?: ExpenseAllocation;
    payerUserId?: string;
    onAllocationChange?: (allocation: ExpenseAllocation) => void;
    onPayerChange?: (payerUserId: string) => void;
} = {}) {
    render(
        <MemoryRouter initialEntries={["/expense-form"]}>
            <Routes>
                <Route
                    path="/expense-form"
                    element={
                        <ExpenseSimpleSplitControl
                            allocation={allocation}
                            amountDigits={2}
                            calculation={calculateExpenseAllocation(
                                "10.00",
                                2,
                                allocation,
                                "CAD"
                            )}
                            currency="CAD"
                            currentUserId="current"
                            groupMembers={members}
                            onAllocationChange={onAllocationChange}
                            onPayerChange={onPayerChange}
                            payerUserId={payerUserId}
                            splitTo="/expense-form/split"
                        />
                    }
                />
                <Route path="/expense-form/split" element={<p>Advanced split</p>} />
            </Routes>
        </MemoryRouter>
    );
    return { onAllocationChange, onPayerChange };
}

describe("ExpenseSimpleSplitControl", () => {
    afterEach(cleanup);

    it("offers the legacy two-person shortcuts and applies a full-amount choice", () => {
        const { onAllocationChange, onPayerChange } = renderControl();

        fireEvent.click(screen.getByRole("button", { name: "Split rule" }));
        expect(
            screen.getByRole("option", { name: "You are owed the full amount" })
        ).toBeInTheDocument();
        expect(
            screen.getByRole("option", { name: "Alice is owed the full amount" })
        ).toBeInTheDocument();
        fireEvent.click(
            screen.getByRole("option", { name: "Alice is owed the full amount" })
        );

        expect(onPayerChange).toHaveBeenCalledWith("other");
        expect(onAllocationChange).toHaveBeenCalledWith({
            mode: "equal",
            participants: [{ userId: "other" }],
        });
    });

    it("opens the dedicated split page only when Unequally is selected", () => {
        renderControl();

        fireEvent.click(screen.getByRole("button", { name: "Split rule" }));
        fireEvent.click(screen.getByRole("option", { name: /Unequally/ }));

        expect(screen.getByText("Advanced split")).toBeInTheDocument();
    });

    it("summarizes an existing advanced allocation instead of replacing it", () => {
        const allocation: ExpenseAllocation = {
            mode: "exact",
            participants: [
                { userId: "other", amount: "6.00" },
                { userId: "current", amount: "4.00" },
            ],
        };
        const { onAllocationChange, onPayerChange } = renderControl({
            allocation,
        });

        expect(screen.getByRole("button", { name: "Split rule" })).toHaveTextContent(
            "Unequally"
        );
        expect(screen.getByText("Exact amounts · 2 people")).toBeInTheDocument();
        expect(onAllocationChange).not.toHaveBeenCalled();
        expect(onPayerChange).not.toHaveBeenCalled();
    });

    it("opens the three-person split picker above the trigger on mobile", () => {
        const allocation: ExpenseAllocation = {
            mode: "equal",
            participants: threeMembers.map(({ userId }) => ({ userId })),
        };
        render(
            <MemoryRouter>
                <ExpenseSimpleSplitControl
                    allocation={allocation}
                    amountDigits={2}
                    calculation={calculateExpenseAllocation(
                        "10.00",
                        2,
                        allocation,
                        "CAD"
                    )}
                    currency="CAD"
                    currentUserId="current"
                    groupMembers={threeMembers}
                    onAllocationChange={vi.fn()}
                    onPayerChange={vi.fn()}
                    payerUserId="current"
                    splitTo="/expense-form/split"
                />
            </MemoryRouter>
        );

        fireEvent.click(screen.getByRole("button", { name: "Split" }));

        expect(screen.getByRole("listbox", { name: "Split options" })).toHaveClass(
            "bottom-full"
        );
    });
});
