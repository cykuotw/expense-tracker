import { fireEvent, render, screen, within } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import Home from "./Home";

const { homeMock } = vi.hoisted(() => ({
    homeMock: vi.fn(),
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

describe("Home mobile summary", () => {
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
                    currency: "NTD",
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
            within(mobileSummary).getByText("You are owed 30 NTD")
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
});
