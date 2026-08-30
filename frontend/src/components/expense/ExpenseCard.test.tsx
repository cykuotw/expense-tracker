import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import ExpenseCard from "./ExpenseCard";

describe("ExpenseCard", () => {
    it("uses the expense-detail route and exposes an interactive card treatment", () => {
        render(
            <MemoryRouter>
                <ExpenseCard
                    expenseId="expense-1"
                    description="Dinner"
                    total="25.50"
                    currency="CAD"
                    expenseTime="2026-08-29T23:33:02Z"
                    currentUser="user-1"
                    isSettled={false}
                    payerUserIds={["user-1"]}
                    payerUsernames={["Current"]}
                    expenseTypeId="type-1"
                    expenseType="Dining out"
                    expenseCategory="Food and Drink"
                />
            </MemoryRouter>
        );

        const card = screen.getByRole("link", { name: "Open expense: Dinner" });
        expect(card).toHaveAttribute("href", "/expense/expense-1");
        expect(card).toHaveClass("hover:-translate-y-0.5", "hover:shadow-lg");
        expect(card).toHaveTextContent("$25.50 CAD");
        expect(card).toHaveTextContent("Dinner");
        expect(card).toHaveTextContent("Paid by Current");
        expect(card).toHaveTextContent("Total");
        expect(card).not.toHaveTextContent("2026-08-29T23:33:02Z");
    });
});
