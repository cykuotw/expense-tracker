import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GroupDetailProvider } from "./GroupDetailContext";
import { useGroupDetail } from "../hooks/GroupDetailContextHooks";

const { apiFetchMock, toastErrorMock, toastSuccessMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
    useParams: () => ({ id: "group-1" }),
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
    toast: { error: toastErrorMock, success: toastSuccessMock },
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function GroupDetailHarness() {
    const context = useGroupDetail();

    return (
        <>
            <output data-testid="loading">
                {context.loading ? "loading" : "idle"}
            </output>
            <output data-testid="expense-order">{context.expenseOrder}</output>
            <output data-testid="unsettled-count">
                {context.unsettledExpenses.length}
            </output>
            <output data-testid="unsettled-has-more">
                {context.unsettledHasMore ? "yes" : "no"}
            </output>
            <output data-testid="settlement-pending">
                {context.settlementPending ? "pending" : "idle"}
            </output>
            <button type="button" onClick={() => context.setExpenseOrder("oldest")}>
                Oldest first
            </button>
            <button type="button" onClick={context.loadMoreUnsettledExpenses}>
                Load more unsettled
            </button>
            <button
                type="button"
                disabled={context.settlementPending}
                onClick={context.handleSettle}
            >
                Settle
            </button>
        </>
    );
}

describe("GroupDetailProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/group_overview/group-1/0?order=")) {
                return Promise.resolve(
                    jsonResponse({
                        group: { groupName: "Group", description: "", currency: "CAD", members: [] },
                        balance: { currency: "CAD", currentUser: "user-1", balances: [] },
                        expenses: { expenses: [], hasMore: false },
                    })
                );
            }
            if (path === "/settle_expense/group-1") {
                return Promise.resolve(
                    jsonResponse({ error: "settlement not permitted" }, 403)
                );
            }
            throw new Error(`Unexpected path: ${path}`);
        });
    });

    afterEach(() => {
        cleanup();
    });

    it("shows the settlement error without running success behavior", async () => {
        const successLog = vi.spyOn(console, "log").mockImplementation(() => {});
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("loading")).toHaveTextContent("idle");
        });

        expect(screen.getByTestId("expense-order")).toHaveTextContent("newest");
        expect(apiFetchMock).toHaveBeenCalledWith(
            "/group_overview/group-1/0?order=newest&status=unsettled",
            expect.objectContaining({ signal: expect.any(AbortSignal) }),
        );

        fireEvent.click(screen.getByRole("button", { name: "Settle" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "settlement not permitted"
            );
        });
        expect(successLog).not.toHaveBeenCalledWith("Settlement successful");
        expect(
            apiFetchMock.mock.calls.filter(
                ([path]) => path === "/settle_expense/group-1"
            )
        ).toHaveLength(1);
        successLog.mockRestore();
    });

    it("refreshes group data after a successful settlement without reloading the page", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/group_overview/group-1/0?order=")) {
                return Promise.resolve(jsonResponse({ group: { groupName: "Group", description: "", currency: "CAD", members: [] }, balance: { currency: "CAD", currentUser: "user-1", balances: [] }, expenses: { expenses: [], hasMore: false } }));
            }
            if (path === "/settle_expense/group-1") return Promise.resolve(jsonResponse({}));
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));
        apiFetchMock.mockClear();

        fireEvent.click(screen.getByRole("button", { name: "Settle" }));

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith("/settle_expense/group-1", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
            });
            expect(apiFetchMock).toHaveBeenCalledWith("/group_overview/group-1/0?order=newest&status=unsettled");
        });
    });

    it("guards duplicate submissions and exposes a pending state", async () => {
        let resolveSettlement!: (response: Response) => void;
        const settlementResponse = new Promise<Response>((resolve) => {
            resolveSettlement = resolve;
        });
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/group_overview/group-1/0?order=")) {
                return Promise.resolve(jsonResponse({ group: { groupName: "Group", description: "", currency: "CAD", members: [] }, balance: { currency: "CAD", currentUser: "user-1", balances: [] }, expenses: { expenses: [], hasMore: false } }));
            }
            if (path === "/settle_expense/group-1") return settlementResponse;
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));
        apiFetchMock.mockClear();

        const settleButton = screen.getByRole("button", { name: "Settle" });
        fireEvent.click(settleButton);
        fireEvent.click(settleButton);

        expect(apiFetchMock).toHaveBeenCalledTimes(1);
        expect(settleButton).toBeDisabled();
        expect(screen.getByTestId("settlement-pending")).toHaveTextContent("pending");

        resolveSettlement(jsonResponse({}));
        await waitFor(() => expect(screen.getByTestId("settlement-pending")).toHaveTextContent("idle"));
    });

    it("confirms an ambiguous response from authoritative state", async () => {
        let overviewCalls = 0;
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/group_overview/group-1/0?order=")) {
                overviewCalls += 1;
                return Promise.resolve(jsonResponse({
                    group: { groupName: "Group", description: "", currency: "CAD", members: [] },
                    balance: { currency: "CAD", currentUser: "user-1", balances: overviewCalls === 1 ? [{ id: "balance-1" }] : [] },
                    expenses: { expenses: overviewCalls === 1 ? [{ expenseId: "expense-1" }] : [], hasMore: false },
                }));
            }
            if (path === "/settle_expense/group-1") return Promise.reject(new TypeError("network lost"));
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));

        fireEvent.click(screen.getByRole("button", { name: "Settle" }));

        await waitFor(() => {
            expect(toastSuccessMock).toHaveBeenCalledWith("Settlement confirmed.");
            expect(screen.getByTestId("unsettled-count")).toHaveTextContent("0");
        });
        expect(toastErrorMock).not.toHaveBeenCalled();
    });

    it("keeps the latest state and offers a safe retry when an ambiguous response is not committed", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/group_overview/group-1/0?order=")) {
                return Promise.resolve(jsonResponse({
                    group: { groupName: "Group", description: "", currency: "CAD", members: [] },
                    balance: { currency: "CAD", currentUser: "user-1", balances: [{ id: "balance-1" }] },
                    expenses: { expenses: [{ expenseId: "expense-1" }], hasMore: false },
                }));
            }
            if (path === "/settle_expense/group-1") return Promise.reject(new TypeError("network lost"));
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));

        fireEvent.click(screen.getByRole("button", { name: "Settle" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Settlement was not confirmed. Review the latest balances before trying again."
            );
            expect(screen.getByTestId("unsettled-count")).toHaveTextContent("1");
        });
        expect(toastSuccessMock).not.toHaveBeenCalled();
    });

    it("reloads expenses in the order selected by the user", async () => {
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );
        await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));
        apiFetchMock.mockClear();

        fireEvent.click(screen.getByRole("button", { name: "Oldest first" }));

        await waitFor(() => {
            expect(screen.getByTestId("expense-order")).toHaveTextContent("oldest");
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/group_overview/group-1/0?order=oldest&status=unsettled",
                expect.objectContaining({ signal: expect.any(AbortSignal) }),
            );
        });
    });

    it("uses the server hasMore flag when loading another unsettled page", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/group_overview/group-1/0?order=newest&status=unsettled") {
                return Promise.resolve(jsonResponse({ group: { groupName: "Group", description: "", currency: "CAD", members: [] }, balance: { currency: "CAD", currentUser: "user-1", balances: [] }, expenses: { expenses: [{ expenseId: "expense-1" }], hasMore: true } }));
            }
            if (path === "/expense_list/group-1/1?order=newest&status=unsettled") {
                return Promise.resolve(jsonResponse({ expenses: [{ expenseId: "expense-2" }], hasMore: false }));
            }
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <GroupDetailProvider>
                <GroupDetailHarness />
            </GroupDetailProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("unsettled-count")).toHaveTextContent("1");
            expect(screen.getByTestId("unsettled-has-more")).toHaveTextContent("yes");
        });
        fireEvent.click(screen.getByRole("button", { name: "Load more unsettled" }));

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/expense_list/group-1/1?order=newest&status=unsettled"
            );
            expect(screen.getByTestId("unsettled-count")).toHaveTextContent("2");
            expect(screen.getByTestId("unsettled-has-more")).toHaveTextContent("no");
        });
    });
});
