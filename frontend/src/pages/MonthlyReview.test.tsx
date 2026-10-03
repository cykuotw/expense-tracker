import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MonthlyReview from "./MonthlyReview";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));

vi.mock("../lib/api", async (importOriginal) => {
    const actual = await importOriginal<typeof import("../lib/api")>();
    return { ...actual, apiFetch: apiFetchMock };
});

vi.mock("../hooks/useCurrencies", () => ({
    useCurrencies: () => ({
        currencies: [
            { code: "CAD", amountDigits: 2 },
            { code: "TWD", amountDigits: 0 },
        ],
        loading: false,
        error: false,
        reload: vi.fn(),
    }),
}));

function renderReview() {
    return render(
        <MemoryRouter initialEntries={["/group/group-1/monthly-review/2026-08"]}>
            <Routes>
                <Route path="/group/:id/monthly-review/:month" element={<MonthlyReview />} />
            </Routes>
            <LocationProbe />
        </MemoryRouter>,
    );
}

function LocationProbe() {
    return <output data-testid="review-location">{useLocation().pathname}</output>;
}

const trendResponse = {
    groupId: "group-1",
    groupName: "Home",
    startMonth: "2026-01",
    endMonth: "2026-08",
    latestReportMonth: "2026-08",
    currencies: [{
        currency: "CAD",
        months: [
            { month: "2026-01", total: "0", expenseCount: 0 },
            { month: "2026-02", total: "0", expenseCount: 0 },
            { month: "2026-03", total: "0", expenseCount: 0 },
            { month: "2026-04", total: "12", expenseCount: 1 },
            { month: "2026-05", total: "0", expenseCount: 0 },
            { month: "2026-06", total: "18", expenseCount: 1 },
            { month: "2026-07", total: "20", expenseCount: 1 },
            { month: "2026-08", total: "30", expenseCount: 1 },
        ],
    }],
};

function mockMonthlyReviewResponses(reviewResponse: object) {
    apiFetchMock.mockImplementation((path = "") => Promise.resolve(new Response(JSON.stringify(
        String(path).includes("monthly-review-trend") ? trendResponse
            : String(path).includes("/expenses?") ? { expenses: [], nextCursor: undefined }
                : reviewResponse,
    ), { status: 200, headers: { "Content-Type": "application/json" } })));
}

function openSortMenu(container: HTMLElement) {
    fireEvent.pointerDown(within(container).getByRole("button", { name: /^Sort expenses:/ }), { button: 0, ctrlKey: false, pointerType: "mouse" });
}

describe("MonthlyReview", () => {
    beforeEach(() => apiFetchMock.mockReset());
    afterEach(cleanup);

    it("renders live currency-separated summaries and all settlement states", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "published",
            publishedAt: "2026-09-01T05:15:00Z",
            currencies: [
                {
                    currency: "CAD",
                    total: "30",
                    expenseCount: 1,
                    categories: [{ name: "Food and Drink", amount: "30" }],
                    payers: [{ userId: "alice", username: "Alice", amount: "30" }],
                    memberNetTotals: [
                        { userId: "alice", username: "Alice", amount: "15" },
                        { userId: "bob", username: "Bob", amount: "-15" },
                    ],
                },
                {
                    currency: "TWD",
                    total: "100",
                    expenseCount: 0,
                    categories: [{ name: "Transportation", amount: "100" }],
                    payers: [],
                    memberNetTotals: [],
                },
            ],
        });

        renderReview();

        expect((await screen.findAllByRole("heading", { name: "Home" })).length).toBeGreaterThan(0);
        expect(screen.getAllByText("30.00 CAD").length).toBeGreaterThan(0);
        expect(screen.getAllByText("100 TWD").length).toBeGreaterThan(0);
        expect(screen.getByRole("img", { name: "Food and Drink: 100%" })).toBeVisible();
        expect(screen.getByRole("img", { name: "Alice: 100%" })).toBeVisible();
        const trendChart = screen.getByRole("group", { name: "CAD monthly spending trend" });
        expect(trendChart).toBeVisible();
        expect(trendChart).toHaveClass("grid-cols-6", "md:grid-cols-12");
        expect(within(trendChart).getByText("30")).toHaveAttribute("title", "30.00 CAD");
        expect(screen.getByRole("link", { name: /August 2026: 30\.00 CAD/ })).toHaveAttribute(
            "href",
            "/group/group-1/monthly-review/2026-08",
        );
        expect(screen.getAllByText("Expenses")[0].closest("details")).not.toHaveAttribute("open");
        expect(apiFetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/expenses?"));
        const memberCards = screen.getAllByRole("heading", { name: "Member net totals" })
            .map((heading) => heading.closest("[data-slot=card]"))
            .filter((card): card is HTMLElement => card !== null);
        expect(memberCards.some((card) => within(card).queryByText("-15.00 CAD") !== null)).toBe(true);
    });

    it("loads expense details only when expanded and paginates on demand", async () => {
        const reviewResponse = {
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "published",
            currencies: [{
                currency: "CAD",
                total: "50",
                expenseCount: 2,
                categories: [{ name: "Food and Drink", amount: "50" }],
                payers: [{ userId: "alice", username: "Alice", amount: "50" }],
                memberNetTotals: [],
            }],
        };
        let expensePage = 0;
        apiFetchMock.mockImplementation((path = "") => {
            const requestPath = String(path);
            if (requestPath.includes("monthly-review-trend")) return Promise.resolve(new Response(JSON.stringify(trendResponse), { status: 200 }));
            if (!requestPath.includes("/expenses?")) return Promise.resolve(new Response(JSON.stringify(reviewResponse), { status: 200 }));
            expensePage += 1;
            const response = expensePage === 1
                ? { expenses: [{ id: "expense-1", description: "Groceries", occurredOn: "2026-08-20", expenseType: "Groceries", category: "Food and Drink", payerId: "alice", payerName: "Alice", total: "30", settled: false }], nextCursor: "next-page" }
                : { expenses: [{ id: "expense-2", description: "Dinner", occurredOn: "2026-08-19", expenseType: "Dining", category: "Food and Drink", payerId: "alice", payerName: "Alice", total: "20", settled: true }] };
            return Promise.resolve(new Response(JSON.stringify(response), { status: 200 }));
        });

        renderReview();
        const disclosure = (await screen.findByText("Expenses")).closest("details");
        expect(disclosure).not.toHaveAttribute("open");
        expect(apiFetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/expenses?"));

        fireEvent.click(within(disclosure as HTMLElement).getByText("Expenses"));
        expect(await screen.findByRole("link", { name: "Open expense: Groceries" })).toHaveAttribute("href", "/expense/expense-1");
        expect(apiFetchMock).toHaveBeenCalledWith("/group/group-1/monthly-review/2026-08/expenses?currency=CAD");

        fireEvent.click(screen.getByRole("button", { name: "Load more expenses" }));
        expect(await screen.findByRole("link", { name: "Open expense: Dinner" })).toHaveAttribute("href", "/expense/expense-2");
        expect(apiFetchMock).toHaveBeenCalledWith("/group/group-1/monthly-review/2026-08/expenses?currency=CAD&cursor=next-page");
        expect(screen.queryByRole("button", { name: "Load more expenses" })).not.toBeInTheDocument();
    });

    it("expands categories independently, sorts the complete list, and resets pagination", async () => {
        const review = { groupId: "group-1", groupName: "Home", month: "2026-08", state: "published", currencies: [{
            currency: "CAD", total: "50", expenseCount: 2,
            categories: [{ name: "Food and Drink", amount: "30" }, { name: "Transportation", amount: "20" }],
            payers: [], memberNetTotals: [],
        }] };
        const expense = (id: string) => ({ id, description: id, occurredOn: "2026-08-20", expenseType: "Groceries", category: "Food and Drink", payerId: "alice", payerName: "Alice", total: "30", settled: false });
        apiFetchMock.mockImplementation((path = "") => {
            if (path.includes("monthly-review-trend")) return Promise.resolve(Response.json(trendResponse));
            if (!path.includes("/expenses?")) return Promise.resolve(Response.json(review));
            const query = new URLSearchParams(path.split("?")[1]);
            return Promise.resolve(Response.json({ expenses: [expense(query.get("cursor") ? "Second page" : query.get("sort") ?? "default")], nextCursor: query.get("cursor") ? undefined : "next-page" }));
        });
        renderReview();
        const food = (await screen.findByText("Food and Drink")).closest("details") as HTMLElement;
        const transport = screen.getByText("Transportation").closest("details") as HTMLElement;
        expect(food).not.toHaveAttribute("open");
        expect(screen.getByRole("img", { name: "Food and Drink: 60%" })).toBeVisible();
        expect(apiFetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/expenses?"));
        fireEvent.click(within(food).getByText("Food and Drink"));
        await within(food).findByRole("link", { name: "Open expense: date_desc" });
        expect(transport).not.toHaveAttribute("open");
        fireEvent.click(within(food).getByRole("button", { name: "Load more expenses" }));
        await within(food).findByRole("link", { name: "Open expense: Second page" });
        expect(apiFetchMock).toHaveBeenCalledWith("/group/group-1/monthly-review/2026-08/expenses?currency=CAD&category=Food+and+Drink&sort=date_desc&cursor=next-page");
        for (const sort of ["date_asc", "amount_desc", "amount_asc"]) {
            openSortMenu(food);
            const labels: Record<string, string> = { date_asc: "Oldest first", amount_desc: "Highest amount", amount_asc: "Lowest amount" };
            fireEvent.click(await screen.findByRole("menuitemradio", { name: labels[sort] }));
            await within(food).findByRole("link", { name: `Open expense: ${sort}` });
            expect(within(food).queryByRole("link", { name: "Open expense: Second page" })).not.toBeInTheDocument();
            expect(apiFetchMock).toHaveBeenLastCalledWith(`/group/group-1/monthly-review/2026-08/expenses?currency=CAD&category=Food+and+Drink&sort=${sort}`);
        }
        const calls = apiFetchMock.mock.calls.length;
        fireEvent.click(within(food).getByText("Food and Drink"));
        fireEvent.click(within(food).getByText("Food and Drink"));
        await waitFor(() => expect(food).toHaveAttribute("open"));
        expect(apiFetchMock).toHaveBeenCalledTimes(calls);
    });

    it("ignores stale sort responses and retries failed category reads", async () => {
        let resolveStale!: (response: Response) => void;
        let fail = true;
        apiFetchMock.mockImplementation((path = "") => {
            if (path.includes("monthly-review-trend")) return Promise.resolve(Response.json(trendResponse));
            if (!path.includes("/expenses?")) return Promise.resolve(Response.json({ groupId: "group-1", groupName: "Home", month: "2026-08", state: "published", currencies: [{ currency: "CAD", total: "30", expenseCount: 1, categories: [{ name: "Food and Drink", amount: "30" }], payers: [], memberNetTotals: [] }] }));
            if (path.includes("sort=date_desc")) return new Promise<Response>((resolve) => { resolveStale = resolve; });
            return Promise.resolve(fail ? new Response("", { status: 500 }) : Response.json({ expenses: [] }));
        });
        renderReview();
        const food = (await screen.findByText("Food and Drink")).closest("details") as HTMLElement;
        fireEvent.click(within(food).getByText("Food and Drink"));
        await within(food).findByRole("button", { name: "Sort expenses: Newest first" });
        await waitFor(() => expect(resolveStale).toBeDefined());
        openSortMenu(food);
        expect(await screen.findByRole("menuitemradio", { name: "Newest first" })).toHaveAttribute("aria-checked", "true");
        fireEvent.click(screen.getByRole("menuitemradio", { name: "Highest amount" }));
        expect(await within(food).findByRole("alert")).toHaveTextContent("Expenses could not be loaded.");
        fail = false;
        fireEvent.click(within(food).getByRole("button", { name: "Try again" }));
        expect(await within(food).findByText("No expenses available.")).toBeVisible();
        await act(async () => resolveStale(Response.json({ expenses: [{ id: "stale", description: "Stale expense", occurredOn: "2026-08-20", total: "30", category: "Food and Drink", expenseType: "Groceries", payerName: "Alice", settled: false }], nextCursor: "stale-cursor" })));
        expect(within(food).queryByRole("link", { name: "Open expense: Stale expense" })).not.toBeInTheDocument();
        expect(within(food).queryByRole("button", { name: "Load more expenses" })).not.toBeInTheDocument();
    });

    it("renders a clear empty state", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "empty",
            currencies: [],
        });

        renderReview();
        expect(await screen.findByRole("heading", { name: "No expenses for this month" })).toBeVisible();
    });

    it("jumps directly to a month selected with the month picker", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "empty",
            currencies: [],
        });

        renderReview();
        const picker = await screen.findByRole("button", { name: "Review month" });
        expect(screen.getByRole("navigation", { name: "Review month navigation" })).toHaveClass(
            "flex-col",
        );
        expect(picker).toHaveClass("min-h-14", "w-full");
        expect(picker).toHaveTextContent("August 2026");
        fireEvent.click(picker);
        fireEvent.click(screen.getByRole("button", { name: "January 2026" }));

        expect(screen.getByTestId("review-location")).toHaveTextContent(
            "/group/group-1/monthly-review/2026-01",
        );
        expect(picker).toHaveTextContent("January 2026");
    });

    it("returns to the latest published report", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "empty",
            currencies: [],
        });

        renderReview();
        await screen.findByRole("heading", { name: "No expenses for this month" });
        fireEvent.click(screen.getByRole("button", { name: "Review month" }));
        fireEvent.click(screen.getByRole("button", { name: "July 2026" }));
        fireEvent.click(await screen.findByRole("button", { name: "Latest report" }));

        expect(screen.getByTestId("review-location")).toHaveTextContent(
            "/group/group-1/monthly-review/2026-08",
        );
    });

    it("uses an overlapping January to June mobile history window at the launch boundary", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "empty",
            currencies: [],
        });

        renderReview();
        const januaryLink = await screen.findByRole("link", {
            name: /January 2026: 0\.00 CAD/,
            hidden: true,
        });
        const augustLink = screen.getByRole("link", { name: /August 2026: 30\.00 CAD/ });
        expect(januaryLink.parentElement).toHaveClass("hidden", "md:flex");
        expect(augustLink.parentElement).toHaveClass("flex");

        fireEvent.click(screen.getByRole("button", { name: "Previous six months" }));

        expect(januaryLink.parentElement).toHaveClass("flex");
        expect(augustLink.parentElement).toHaveClass("hidden", "md:flex");
        expect(screen.getByText("January 2026–June 2026")).toBeVisible();
    });

    it("shows compact matching labels in mobile review navigation", async () => {
        mockMonthlyReviewResponses({
            groupId: "group-1",
            groupName: "Home",
            month: "2026-08",
            state: "empty",
            currencies: [],
        });

        renderReview();
        const navigation = await screen.findByRole("navigation", { name: "Review month navigation" });
        expect(within(navigation).getByText("Prev")).toHaveClass("sm:hidden");
        expect(within(navigation).getByText("Latest")).toHaveClass("sm:hidden");
        expect(within(navigation).getByText("Latest report")).toHaveClass("hidden", "sm:inline");
        expect(within(navigation).getByText("Next")).toBeVisible();
    });
});
