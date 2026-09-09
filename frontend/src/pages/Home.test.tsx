import { fireEvent, render, screen, within } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import Home from "./Home";

const { homeMock, currenciesMock } = vi.hoisted(() => ({
    homeMock: vi.fn(),
    currenciesMock: vi.fn(),
}));

vi.mock("../contexts/HomeContextHooks", async (importOriginal) => {
    const actual = await importOriginal<
        typeof import("../contexts/HomeContextHooks")
    >();
    return {
        ...actual,
        useHome: () => homeMock(),
    };
});

vi.mock("../contexts/HomeContext", () => ({
    HomeProvider: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("../hooks/useCurrencies", () => ({
    useCurrencies: () => currenciesMock(),
}));

describe("Home mobile summary", () => {
    currenciesMock.mockReturnValue({
        currencies: [
            { code: "CAD", amountDigits: 2 },
            { code: "USD", amountDigits: 2 },
            { code: "TWD", amountDigits: 0 },
        ],
        loading: false,
        error: false,
        reload: vi.fn(),
    });

    it("keeps the summary compact above the group cards and expands extra balances", () => {
        homeMock.mockReturnValue({
            loading: false,
            groupCards: [
                {
                    id: "group-1",
                    groupName: "Trip",
                    description: "",
                    currency: "CAD",
                    balanceStatus: "owed",
                    balanceAmount: "10",
                },
                {
                    id: "group-2",
                    groupName: "House",
                    description: "",
                    currency: "USD",
                    balanceStatus: "owing",
                    balanceAmount: "20",
                },
                {
                    id: "group-3",
                    groupName: "Dinner",
                    description: "",
                    currency: "TWD",
                    balanceStatus: "owed",
                    balanceAmount: "30",
                },
            ],
        });

        render(
            <MemoryRouter>
                <Home />
            </MemoryRouter>
        );

        const mobileSummary = screen.getByTestId("mobile-home-summary");
        expect(mobileSummary).toHaveTextContent("3 active groups");
        expect(mobileSummary).toHaveTextContent("3 unsettled");
        expect(
            within(mobileSummary).getByText("You are owed 10.00 CAD")
        ).toBeVisible();
        expect(
            within(mobileSummary).getByText("You are owed 30 TWD")
        ).toBeVisible();
        expect(
            within(mobileSummary).queryByText("You owe 20.00 USD")
        ).toBeNull();
        expect(screen.getByTestId("home-summary-panel")).toHaveClass(
            "hidden",
            "sm:block"
        );
        expect(within(screen.getByTestId("group-card-list")).getAllByRole("link")).toHaveLength(3);

        fireEvent.click(screen.getByRole("button", { name: "View all 3 balances" }));

        expect(
            within(mobileSummary).getByText("You owe 20.00 USD")
        ).toBeVisible();
        expect(
            screen.getByRole("button", { name: "Show fewer balances" })
        ).toHaveAttribute("aria-expanded", "true");
    });

    it("does not report all settled when currency metadata is unavailable", () => {
        currenciesMock.mockReturnValue({
            currencies: [],
            loading: false,
            error: true,
            reload: vi.fn(),
        });
        homeMock.mockReturnValue({
            loading: false,
            groupCards: [{
                id: "group-1",
                groupName: "Trip",
                description: "",
                currency: "CAD",
                balanceStatus: "settled",
                balanceAmount: "0",
            }],
        });

        render(<MemoryRouter><Home /></MemoryRouter>);

        expect(screen.getByText(/Balance totals unavailable/)).toBeVisible();
        expect(screen.queryByText("All settled")).toBeNull();
    });
});
