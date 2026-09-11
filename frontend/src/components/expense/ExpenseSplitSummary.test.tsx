import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import ExpenseSplitSummary from "./ExpenseSplitSummary";

describe("ExpenseSplitSummary", () => {
    it("renders the saved mode, participants, and calculated shares", () => {
        render(
            <MemoryRouter>
                <ExpenseSplitSummary
                    allocation={{
                        mode: "adjustment",
                        participants: [
                            { userId: "user-a", amount: "1.00" },
                            { userId: "user-b", amount: "0" },
                        ],
                    }}
                    amountDigits={2}
                    calculation={{
                        valid: true,
                        message: "Split total is complete.",
                        remainingUnits: 0n,
                        shares: new Map([
                            ["user-a", 550n],
                            ["user-b", 450n],
                        ]),
                    }}
                    currency="CAD"
                    groupMembers={[
                        { userId: "user-a", username: "Alice" },
                        { userId: "user-b", username: "Bob" },
                    ]}
                    to="split"
                />
            </MemoryRouter>
        );

        expect(screen.getByText("Equal + adjustments · 2 people")).toBeInTheDocument();
        expect(screen.getByText("Alice")).toBeInTheDocument();
        expect(screen.getByText("5.50 CAD")).toBeInTheDocument();
        expect(screen.getByText("Bob")).toBeInTheDocument();
        expect(screen.getByText("4.50 CAD")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Edit split/ })).toHaveAttribute(
            "href",
            "/split"
        );
    });
});
