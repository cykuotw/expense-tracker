import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CreateExpenseProvider } from "./CreateExpenseContext";
import { useCreateExpense } from "../hooks/CreateExpenseContextHooks";

const { apiFetchMock, navigateMock, toastErrorMock, toastSuccessMock } =
    vi.hoisted(() => ({
        apiFetchMock: vi.fn(),
        navigateMock: vi.fn(),
        toastErrorMock: vi.fn(),
        toastSuccessMock: vi.fn(),
    }));

vi.mock("react-router-dom", () => ({
    useNavigate: () => navigateMock,
    useSearchParams: () => [new URLSearchParams("g=group-1")],
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
    toast: {
        error: toastErrorMock,
        success: toastSuccessMock,
    },
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function CreateExpenseHarness() {
    const context = useCreateExpense();

    return (
        <form aria-label="expense form" onSubmit={context.handleCreateExpense}>
            <input
                aria-label="amount"
                type="number"
                value={context.totalInput}
                onChange={(event) => context.setTotalInput(event.target.value)}
            />
            <output data-testid="total">{context.total}</output>
            <output data-testid="members">{context.ledgers.length}</output>
            <output data-testid="member-load-status">
                {context.groupMembersLoadStatus}
            </output>
            <output data-testid="selected-group">
                {context.selectedGroupId}
            </output>
            <button
                type="button"
                onClick={() => context.setSelectedGroupId("group-2")}
            >
                Select second group
            </button>
            <button type="button" onClick={context.reloadGroupMembers}>
                Retry members
            </button>
            <input
                aria-label="expense date"
                type="date"
                value={context.occurredOn}
                onChange={(event) => context.setOccurredOn(event.target.value)}
            />
            <output data-testid="indicator">
                {context.indicatorShow ? "loading" : "idle"}
            </output>
            <button type="submit">Create</button>
        </form>
    );
}

describe("CreateExpenseProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense_create_options?groupId=group-1") {
                return Promise.resolve(
                    jsonResponse({
                        groups: [
                            {
                                id: "group-1",
                                groupName: "Group",
                                description: "",
                                currency: "CAD",
                            },
                            {
                                id: "group-2",
                                groupName: "Other group",
                                description: "",
                                currency: "USD",
                            },
                        ],
                        expenseTypes: [
                            { id: "type-1", category: "Other", name: "General" },
                        ],
                        currencies: [
                            { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                            { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
                        ],
                        group: {
                            currency: "CAD",
                            members: [
                                { userId: "user-1", username: "Current" },
                            ],
                        },
                    })
                );
            }
            if (path === "/expense_create_options?groupId=group-2") {
                return Promise.resolve(
                    jsonResponse({
                        groups: [],
                        expenseTypes: [],
                        currencies: [
                            { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                            { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
                        ],
                        group: {
                            currency: "USD",
                            members: [
                                { userId: "user-2", username: "Other" },
                                { userId: "user-1", username: "Current" },
                            ],
                        },
                    })
                );
            }
            if (path === "/create_expense") {
                return Promise.reject(new Error("network unavailable"));
            }
            throw new Error(`Unexpected path: ${path}`);
        });
    });

    afterEach(() => {
        cleanup();
    });

    it("loads all initial form options with one page request", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("member-load-status")).toHaveTextContent(
                "ready"
            );
        });
        const initialReads = apiFetchMock.mock.calls.filter(
            ([path]) => path !== "/create_expense"
        );
        expect(initialReads).toHaveLength(1);
        expect(initialReads[0][0]).toBe(
            "/expense_create_options?groupId=group-1"
        );
    });

    it("uses the fallback and clears the indicator after a request rejection", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("members")).toHaveTextContent("1");
        });
        fireEvent.change(screen.getByLabelText("expense date"), {
			target: { value: "2026-08-31" },
		});
        fireEvent.change(screen.getByLabelText("amount"), { target: { value: "10" } });

        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Failed to create expense."
            );
        });
        expect(screen.getByTestId("indicator")).toHaveTextContent("idle");
		const createRequest = apiFetchMock.mock.calls.find(
			([path]) => path === "/create_expense"
		)?.[1] as RequestInit;
		expect(JSON.parse(createRequest.body as string)).toMatchObject({
			occurredOn: "2026-08-31",
		});
        expect(toastSuccessMock).not.toHaveBeenCalled();
        expect(navigateMock).not.toHaveBeenCalled();
    });

    it("reloads members after choosing a different group", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("members")).toHaveTextContent("1");
        });

        fireEvent.click(
            screen.getByRole("button", { name: "Select second group" })
        );

        await waitFor(() => {
            expect(screen.getByTestId("selected-group")).toHaveTextContent(
                "group-2"
            );
            expect(screen.getByTestId("members")).toHaveTextContent("2");
        });
        expect(apiFetchMock).toHaveBeenCalledWith(
            "/expense_create_options?groupId=group-2",
            expect.objectContaining({ method: "GET" })
        );
    });

    it("aborts the split-options request after leaving the form", async () => {
        let groupSignal: AbortSignal | undefined;
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense_create_options?groupId=group-1") {
                    groupSignal = init.signal as AbortSignal;
                    return new Promise(() => undefined);
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );

        const view = render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );
        await waitFor(() => expect(groupSignal).toBeDefined());

        view.unmount();

        expect(groupSignal?.aborted).toBe(true);
        expect(screen.queryByTestId("member-load-status")).toBeNull();
    });

    it("uses one idempotency key for an in-flight submit and an unchanged retry", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("members")).toHaveTextContent("1");
        });

        const form = screen.getByRole("form", { name: "expense form" });
        fireEvent.change(screen.getByLabelText("amount"), { target: { value: "10" } });
        fireEvent.submit(form);
        fireEvent.submit(form);

        await waitFor(() => {
            expect(
                apiFetchMock.mock.calls.filter(([path]) => path === "/create_expense")
            ).toHaveLength(1);
        });
        const firstHeaders = apiFetchMock.mock.calls.find(
            ([path]) => path === "/create_expense"
        )?.[1]?.headers as Record<string, string>;
        expect(firstHeaders["Idempotency-Key"]).toMatch(
            /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
        );

        await waitFor(() => expect(screen.getByTestId("indicator")).toHaveTextContent("idle"));
        fireEvent.submit(form);
        await waitFor(() => {
            expect(
                apiFetchMock.mock.calls.filter(([path]) => path === "/create_expense")
            ).toHaveLength(2);
        });
        const retryHeaders = apiFetchMock.mock.calls.filter(
            ([path]) => path === "/create_expense"
        )[1][1]?.headers as Record<string, string>;
        expect(retryHeaders["Idempotency-Key"]).toBe(firstHeaders["Idempotency-Key"]);
    });

    it("keeps the amount input editable when it is cleared or replaced", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );

        const amountInput = screen.getByLabelText("amount");
        fireEvent.change(amountInput, { target: { value: "12.50" } });
        expect(amountInput).toHaveValue(12.5);
        expect(screen.getByTestId("total")).toHaveTextContent("12.50");

        fireEvent.change(amountInput, { target: { value: "" } });
        expect(amountInput).toHaveValue(null);
        expect(screen.getByTestId("total")).toHaveTextContent("");
    });
});
