import { useState } from "react";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, Link, Outlet, RouterProvider, useParams } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import NavigationProvider from "./NavigationProvider";
import { GroupDetailProvider } from "./GroupDetailContext";
import { useGroupDetail } from "../hooks/GroupDetailContextHooks";
import { useNavigationReady, useNavigationState, useReturnNavigation } from "../hooks/navigation";
import DesktopBackLink from "../components/DesktopBackLink";
import MobilePageHeader from "../components/MobilePageHeader";
import MonthlyReview from "../pages/MonthlyReview";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("../lib/api", async (importOriginal) => ({
    ...await importOriginal<typeof import("../lib/api")>(), apiFetch: apiFetchMock,
}));
vi.mock("../hooks/useCurrencies", () => ({
    useCurrencies: () => ({ currencies: [{ code: "CAD", amountDigits: 2 }], loading: false, error: false, reload: vi.fn() }),
}));

function json(value: unknown) { return new Response(JSON.stringify(value), { status: 200 }); }

function Session() {
    const [actor, setActor] = useState(0);
    return <><button onClick={() => setActor((value) => value + 1)}>Change account</button><NavigationProvider key={actor}><main id="main-content"><Outlet /></main></NavigationProvider></>;
}
function Home() { return <div className="page-shell"><h1>Groups</h1><Link to="/group/g">Open group</Link><Link to="/admin/users">Open admin</Link></div>; }
function GroupView() {
    const state = useGroupDetail();
    const [settled, setSettled] = useNavigationState<boolean>("group.settled", false);
    return <div className="page-shell"><h1>Group</h1>
        <DesktopBackLink to="/" label="Back to groups" />
        <Link to="/group/g/monthly-review/2026-08">Open review</Link>
        <Link to="/admin/users">Open admin</Link>
        <select aria-label="Expense order" value={state.expenseOrder} onChange={(event) => state.setExpenseOrder(event.target.value as "newest" | "oldest")}><option value="newest">Newest</option><option value="oldest">Oldest</option></select>
        <button onClick={() => { setSettled(true); void state.loadSettledExpenses(); }}>Show settled</button>
        <output>{settled ? "Settled visible" : "Settled hidden"}</output>
        {state.unsettledExpenses.map((expense) => <Link key={expense.expenseId} to={`/expense/${expense.expenseId}`}>{expense.description}</Link>)}
        <button onClick={() => void state.loadMoreUnsettledExpenses()}>Load group expenses</button>
    </div>;
}
function Expense() {
    const { id } = useParams();
    return <div className="page-shell"><h1>Expense</h1><MobilePageHeader title="Expense" backTo="/group/g" backLabel="Back to group" /><DesktopBackLink to="/group/g" label="Back to group" /><Link to={`/expense/${id}/edit`}>Edit expense</Link></div>;
}
function Edit() {
    const { id } = useParams();
    const { finish } = useReturnNavigation(`/expense/${id}`);
    return <div className="page-shell"><h1>Edit</h1><DesktopBackLink to={`/expense/${id}`} label="Back to expense" /><button onClick={() => finish(`/expense/${id}`)}>Save changes</button></div>;
}
function Draft() {
    const [draft, setDraft] = useState("");
    return <div className="page-shell"><label>Draft<input value={draft} onChange={(event) => setDraft(event.target.value)} /></label><Outlet /></div>;
}
function Form() { return <><h1>Form</h1><Link to="/create_expense/split">Open split</Link><Link to="/create_expense/receipt">Open receipt</Link></>; }
function Step() { const { finish } = useReturnNavigation("/create_expense"); return <><h1>Step</h1><DesktopBackLink to="/create_expense" label="Cancel changes" /><button onClick={() => finish()}>Apply step</button></>; }
function Admin() { return <div className="page-shell"><h1>Admin</h1><DesktopBackLink to="/account" label="Back to account" /></div>; }
function Delayed() {
    const [ready, setReady] = useState(false);
    useNavigationReady(ready);
    return <div className="page-shell"><h1>Delayed</h1><button onClick={() => setReady(true)}>Content ready</button></div>;
}
function setup(initial = "/") {
    const router = createMemoryRouter([{ element: <Session />, children: [
        { path: "/", element: <Home /> },
        { path: "/group/:id", element: <GroupDetailProvider><GroupView /></GroupDetailProvider> },
        { path: "/group/:id/monthly-review/:month", element: <MonthlyReview /> },
        { path: "/expense/:id", element: <Expense /> },
        { path: "/expense/:id/edit", element: <Edit /> },
        { path: "/create_expense", element: <Draft />, children: [{ index: true, element: <Form /> }, { path: "split", element: <Step /> }, { path: "receipt", element: <Step /> }] },
        { path: "/admin/users", element: <Admin /> },
        { path: "/account", element: <Delayed /> },
    ] }], { initialEntries: [initial] });
    render(<RouterProvider router={router} />);
    return router;
}

describe("shared return journeys", () => {
    beforeEach(() => {
        vi.stubGlobal("scrollTo", vi.fn());
        apiFetchMock.mockImplementation(async (path: string) => {
            if (path.includes("group_overview")) return json({ group: { groupName: "Group", members: [] }, expenses: { expenses: [{ expenseId: "e", description: "First expense" }], hasMore: true } });
            if (path.includes("expense_list")) return json({ expenses: [{ expenseId: "e2", description: "Second expense" }], hasMore: false });
            if (path.includes("monthly-review-trend")) return json({ groupId: "g", groupName: "Group", startMonth: "2026-01", endMonth: "2026-08", currencies: [{ currency: "CAD", months: Array.from({ length: 8 }, (_, index) => ({ month: `2026-0${index + 1}`, total: "10", expenseCount: 1 })) }] });
            if (path.includes("/expenses?")) return json({ expenses: [{ id: path.includes("cursor=next") ? "e2" : "e", description: path.includes("cursor=next") ? "Second review expense" : "First review expense", occurredOn: "2026-08-01", total: "10", currency: "CAD", payerName: "Alex", category: "Food" }], nextCursor: path.includes("cursor=next") ? undefined : "next" });
            return json({ groupId: "g", groupName: "Group", month: "2026-08", state: "published", currencies: [{ currency: "CAD", total: "20", expenseCount: 2, categories: [{ name: "Food", amount: "20" }], payers: [], memberNetTotals: [] }] });
        });
    });
    afterEach(() => { cleanup(); vi.unstubAllGlobals(); apiFetchMock.mockReset(); });

    it("restores group order, loaded pages and visibility using freshly fetched data", async () => {
        const router = setup("/group/g");
        await screen.findByRole("link", { name: "First expense" });
        fireEvent.change(screen.getByLabelText("Expense order"), { target: { value: "oldest" } });
        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(expect.stringContaining("order=oldest"), expect.anything()));
        fireEvent.click(screen.getByText("Show settled"));
        fireEvent.click(screen.getByText("Load group expenses"));
        fireEvent.click(await screen.findByRole("link", { name: "Second expense" }));
        fireEvent.click(screen.getAllByRole("link", { name: "Back to group" })[0]);
        expect(router.state.location.pathname).toBe("/group/g");
        expect(screen.getByLabelText("Expense order")).toHaveValue("oldest");
        expect(screen.getByText("Settled visible")).toBeInTheDocument();
        expect(await screen.findByRole("link", { name: "Second expense" })).toBeInTheDocument();
        expect(apiFetchMock.mock.calls.filter(([path]) => String(path).includes("group_overview")).length).toBeGreaterThanOrEqual(3);
    });

    it("restores a monthly-review disclosure, sort and cursor depth after expense editing", async () => {
        const router = setup("/group/g/monthly-review/2026-08");
        const summary = await screen.findByText("Food");
        fireEvent.click(screen.getByRole("button", { name: "Previous six months" }));
        expect(screen.getByText("January 2026–June 2026")).toBeInTheDocument();
        const details = summary.closest("details")!;
        act(() => { details.open = true; fireEvent(details, new Event("toggle")); });
        await screen.findByRole("link", { name: "Open expense: First review expense" });
        fireEvent.pointerDown(screen.getByRole("button", { name: "Sort expenses: Newest first" }), { button: 0, ctrlKey: false, pointerType: "mouse" });
        fireEvent.click(await screen.findByRole("menuitemradio", { name: "Highest amount" }));
        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(expect.stringContaining("sort=amount_desc")));
        await screen.findByRole("link", { name: "Open expense: First review expense" });
        fireEvent.click(screen.getByRole("button", { name: "Load more expenses" }));
        fireEvent.click(await screen.findByRole("link", { name: "Open expense: Second review expense" }));
        const detailKey = router.state.location.key;
        for (let round = 0; round < 2; round++) {
            fireEvent.click(screen.getByRole("link", { name: "Edit expense" }));
            fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
            expect(router.state.location.key).toBe(detailKey);
            expect(screen.getAllByRole("link", { name: "Back to monthly review" })).toHaveLength(2);
        }
        fireEvent.click(screen.getAllByRole("link", { name: "Back to monthly review" })[0]);
        expect(router.state.location.pathname).toBe("/group/g/monthly-review/2026-08");
        expect((await screen.findByText("Food")).closest("details")).toHaveAttribute("open");
        expect(screen.getByText("January 2026–June 2026")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Sort expenses: Highest amount" })).toBeInTheDocument();
        expect(await screen.findByRole("link", { name: "Open expense: Second review expense" })).toBeInTheDocument();
    });

    it("keeps the main draft alive through repeated child apply and cancel returns", () => {
        const router = setup("/create_expense");
        const originalKey = router.state.location.key;
        fireEvent.change(screen.getByLabelText("Draft"), { target: { value: "Keep this draft" } });
        for (const action of ["Open split", "Open receipt"]) {
            fireEvent.click(screen.getByRole("link", { name: action }));
            fireEvent.click(screen.getByRole("button", { name: "Apply step" }));
            expect(router.state.location.key).toBe(originalKey);
            expect(screen.getByLabelText("Draft")).toHaveValue("Keep this draft");
        }
        fireEvent.click(screen.getByRole("link", { name: "Open split" }));
        fireEvent.click(screen.getByRole("link", { name: "Cancel changes" }));
        expect(router.state.location.key).toBe(originalKey);
    });

    it("respects admin menu entry and clears provenance on account change", async () => {
        setup();
        fireEvent.click(screen.getByRole("link", { name: "Open admin" }));
        expect(screen.getByRole("link", { name: "Back to groups" })).toHaveAttribute("href", "/");
        fireEvent.click(screen.getByRole("button", { name: "Change account" }));
        expect(await screen.findByRole("link", { name: "Back to account" })).toHaveAttribute("href", "/account");
    });

    it("keeps browser Back/Forward independent from route return defaults", async () => {
        const router = setup();
        fireEvent.click(screen.getByRole("link", { name: "Open admin" }));
        await act(() => router.navigate(-1));
        expect(screen.getByRole("heading", { name: "Groups" })).toBeInTheDocument();
        await act(() => router.navigate(1));
        expect(screen.getByRole("heading", { name: "Admin" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Back to groups" })).toBeInTheDocument();
    });

    it("returns admin menu entry to a group rather than its Settings fallback", async () => {
        const router = setup("/group/g");
        await screen.findByRole("link", { name: "First expense" });
        fireEvent.click(screen.getByRole("link", { name: "Open admin" }));
        fireEvent.click(screen.getByRole("link", { name: "Back to group" }));
        expect(router.state.location.pathname).toBe("/group/g");
    });

    it("waits for async content readiness before scrolling", async () => {
        setup("/account");
        await new Promise((resolve) => setTimeout(resolve, 40));
        expect(window.scrollTo).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "Content ready" }));
        await waitFor(() => expect(window.scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: "instant" }));
    });
});
