import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CreateExpenseProvider } from "./CreateExpenseContext";
import { useCreateExpense } from "../hooks/CreateExpenseContextHooks";
import type { ReceiptDraftResult } from "../lib/receiptOcr";

const { apiFetchMock, navigateMock, toastErrorMock, toastSuccessMock, receiptDraftMock } =
    vi.hoisted(() => ({
        apiFetchMock: vi.fn(),
        navigateMock: vi.fn(),
        toastErrorMock: vi.fn(),
        toastSuccessMock: vi.fn(),
        receiptDraftMock: vi.fn(),
    }));

vi.mock("../lib/receiptOcr", async () => ({
    ...await vi.importActual<typeof import("../lib/receiptOcr")>("../lib/receiptOcr"),
    requestReceiptDraftWithRetention: (...args: unknown[]) => receiptDraftMock(...args),
}));

vi.mock("../hooks/AuthContextHooks", () => ({
    useAuth: () => ({ userID: "user-1" }),
}));

vi.mock("react-router-dom", () => ({
    useLocation: () => ({ key: "test", pathname: "/create_expense", search: "?g=group-1", hash: "" }),
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
            <button type="button" disabled={context.receiptOCREnabled !== true} onClick={() => void context.startReceiptOCR({ blob: new Blob(["jpeg"], { type: "image/jpeg" }), width: 10, height: 20, quality: 0.8 })}>Start test scan</button>
            <button type="button" onClick={() => context.clearReceiptWorkflow()}>Discard test scan</button>
            <output data-testid="scan-usage">{context.ocrUsage ? JSON.stringify(context.ocrUsage) : "none"}</output>
            <output data-testid="scan-status">{context.ocrStatus}</output>
            <input
                aria-label="amount"
                type="number"
                value={context.totalInput}
                onChange={(event) => context.setTotalInput(event.target.value)}
            />
            <output data-testid="total">{context.total}</output>
            <output data-testid="members">
                {context.allocation.participants.length}
            </output>
            <output data-testid="participant-ids">
                {context.allocation.participants.map(({ userId }) => userId).join(",")}
            </output>
            <output data-testid="payer">{context.payer}</output>
            <output data-testid="member-load-status">
                {context.groupMembersLoadStatus}
            </output>
            <output data-testid="selected-group">
                {context.selectedGroupId}
            </output>
            <input
                aria-label="description"
                value={context.description}
                onChange={(event) =>
                    context.setDescription(event.target.value)
                }
            />
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
            <output data-testid="submission-error">
                {context.submissionError}
            </output>
            <button
                type="button"
                onClick={() => context.applyReviewedReceipt({
                    merchant: "Cafe",
                    date: "2026-09-27",
                    currencySuggestion: "USD",
                    subtotal: "5.00",
                    tax: "0.65",
                    tip: "1.00",
                    total: "6.65",
                    items: [{
                        id: "item-1", description: "Coffee", quantity: "2",
                        unit: "cup", unitPrice: "2.50", lineTotal: "5.00",
                    }],
                })}
            >
                Apply receipt
            </button>
            <button type="button" onClick={() => context.setCategoryBreakdown([{ expenseTypeId: "type-2", amount: "200", note: "Groceries" }, { expenseTypeId: "type-3", amount: "100", note: "Trolley" }])}>Categorize purchase</button>
            <button type="submit">Create</button>
        </form>
    );
}

describe("CreateExpenseProvider error handling", () => {
    it("clears usage on rescan and discard and ignores a discarded scan completing late", async () => {
        const blank = { value: "" };
        const result: ReceiptDraftResult = {
            draft: { merchant: { value: "Shop" }, date: blank, currencySuggestion: blank, subtotal: blank, tax: blank, tip: blank, total: { value: "10" }, items: [] },
            receiptToken: null, receiptExpiresAt: null,
            usage: { model: "gemini-3.8-flash", tokens: { input: 100, output: 20, thinking: 0, cachedInput: 0, total: 120 }, estimatedCostUsd: 0.00015 },
        };
        let finishLate!: (value: ReceiptDraftResult) => void;
        receiptDraftMock.mockReset().mockResolvedValueOnce(result).mockImplementationOnce(() => new Promise<ReceiptDraftResult>((resolve) => { finishLate = resolve; }));
        render(<CreateExpenseProvider><CreateExpenseHarness /></CreateExpenseProvider>);
        const start = screen.getByRole("button", { name: "Start test scan" });
        await waitFor(() => expect(start).toBeEnabled());
        fireEvent.click(start);
        await waitFor(() => expect(screen.getByTestId("scan-usage")).toHaveTextContent("0.00015"));
        fireEvent.click(start);
        expect(screen.getByTestId("scan-status")).toHaveTextContent("scanning");
        expect(screen.getByTestId("scan-usage")).toHaveTextContent("none");
        fireEvent.click(screen.getByRole("button", { name: "Discard test scan" }));
        finishLate(result);
        await waitFor(() => expect(screen.getByTestId("scan-status")).toHaveTextContent("idle"));
        expect(screen.getByTestId("scan-usage")).toHaveTextContent("none");
        receiptDraftMock.mockResolvedValueOnce({ ...result, usage: undefined });
        fireEvent.click(start);
        await waitFor(() => expect(screen.getByTestId("scan-status")).toHaveTextContent("ready"));
        expect(screen.getByTestId("scan-usage")).toHaveTextContent("none");
    });
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/account") {
                return Promise.resolve(jsonResponse({ capabilities: { receiptOcr: true } }));
            }
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
                            currentUserId: "user-1",
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
                            currentUserId: "user-1",
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

    it("submits one purchase with canonical category amounts and reuses its retry key", async () => {
        render(<CreateExpenseProvider><CreateExpenseHarness /></CreateExpenseProvider>);
        await waitFor(() => expect(screen.getByTestId("member-load-status")).toHaveTextContent("ready"));
        fireEvent.change(screen.getByLabelText("amount"), { target: { value: "300" } });
        fireEvent.change(screen.getByLabelText("description"), { target: { value: "Costco" } });
        fireEvent.click(screen.getByRole("button", { name: "Categorize purchase" }));
        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));
        await waitFor(() => expect(screen.getByTestId("submission-error").textContent).not.toBe(""));
        const first = apiFetchMock.mock.calls.find(([path]) => path === "/create_expense")![1];
        expect(JSON.parse(first.body)).toMatchObject({ expTypeId: "type-2", total: "300.00", categoryBreakdown: [{ expenseTypeId: "type-2", amount: "200.00", note: "Groceries" }, { expenseTypeId: "type-3", amount: "100.00", note: "Trolley" }] });
        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));
        await waitFor(() => expect(apiFetchMock.mock.calls.filter(([path]) => path === "/create_expense")).toHaveLength(2));
        expect(apiFetchMock.mock.calls.filter(([path]) => path === "/create_expense")[1][1].headers["Idempotency-Key"]).toBe(first.headers["Idempotency-Key"]);
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
            ([path]) => path !== "/create_expense" && path !== "/account"
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
        fireEvent.change(screen.getByLabelText("description"), {
            target: { value: "Dinner" },
        });

        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));

        await waitFor(() =>
            expect(screen.getByTestId("submission-error")).toHaveTextContent(
                "We couldn't save this expense. Check your connection and try again."
            )
        );
        expect(screen.getByTestId("indicator")).toHaveTextContent("idle");
		expect(screen.getByLabelText("amount")).toHaveValue(10);
		expect(screen.getByLabelText("description")).toHaveValue("Dinner");
		const createRequest = apiFetchMock.mock.calls.find(
			([path]) => path === "/create_expense"
		)?.[1] as RequestInit;
		expect(JSON.parse(createRequest.body as string)).toMatchObject({
			occurredOn: "2026-08-31",
		});
        expect(toastSuccessMock).not.toHaveBeenCalled();
        expect(toastErrorMock).not.toHaveBeenCalled();
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

    it("keeps the current payer and split until new group options load successfully", async () => {
        let rejectSecondGroup: ((error: Error) => void) | undefined;
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense_create_options?groupId=group-1") {
                return Promise.resolve(
                    jsonResponse({
                        groups: [],
                        expenseTypes: [
                            { id: "type-1", category: "Other", name: "General" },
                        ],
                        currencies: [
                            { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                        ],
                        group: {
                            currentUserId: "user-1",
                            currency: "CAD",
                            members: [{ userId: "user-1", username: "Current" }],
                        },
                    })
                );
            }
            if (path === "/expense_create_options?groupId=group-2") {
                return new Promise((_, reject) => {
                    rejectSecondGroup = reject;
                });
            }
            throw new Error(`Unexpected path: ${path}`);
        });
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("participant-ids")).toHaveTextContent("user-1");
            expect(screen.getByTestId("payer")).toHaveTextContent("user-1");
        });

        fireEvent.click(screen.getByRole("button", { name: "Select second group" }));

        expect(screen.getByTestId("participant-ids")).toHaveTextContent("user-1");
        expect(screen.getByTestId("payer")).toHaveTextContent("user-1");
        rejectSecondGroup?.(new Error("load failed"));
        await waitFor(() => {
            expect(screen.getByTestId("member-load-status")).toHaveTextContent("error");
        });
        expect(screen.getByTestId("participant-ids")).toHaveTextContent("user-1");
        expect(screen.getByTestId("payer")).toHaveTextContent("user-1");
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
        fireEvent.change(screen.getByLabelText("amount"), {
            target: { value: "10" },
        });
        fireEvent.change(screen.getByLabelText("description"), {
            target: { value: "Dinner" },
        });
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

        await waitFor(() =>
            expect(screen.getByTestId("indicator")).toHaveTextContent("idle")
        );
        fireEvent.submit(form);
        await waitFor(() => {
            expect(
                apiFetchMock.mock.calls.filter(
                    ([path]) => path === "/create_expense"
                )
            ).toHaveLength(2);
        });
        const retryHeaders = apiFetchMock.mock.calls.filter(
            ([path]) => path === "/create_expense"
        )[1][1]?.headers as Record<string, string>;
        expect(retryHeaders["Idempotency-Key"]).toBe(firstHeaders["Idempotency-Key"]);
    });

    it("uses a new idempotency key when the visible values change before retry", async () => {
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
        fireEvent.change(screen.getByLabelText("description"), {
            target: { value: "Dinner" },
        });
        fireEvent.submit(form);
        await waitFor(() => expect(screen.getByTestId("indicator")).toHaveTextContent("idle"));

        fireEvent.change(screen.getByLabelText("description"), {
            target: { value: "Updated dinner" },
        });
        fireEvent.submit(form);
        await waitFor(() => {
            expect(
                apiFetchMock.mock.calls.filter(([path]) => path === "/create_expense")
            ).toHaveLength(2);
        });

        const createRequests = apiFetchMock.mock.calls.filter(
            ([path]) => path === "/create_expense"
        );
        const firstHeaders = createRequests[0][1]?.headers as Record<
            string,
            string
        >;
        const retryHeaders = createRequests[1][1]?.headers as Record<
            string,
            string
        >;
        expect(retryHeaders["Idempotency-Key"]).not.toBe(
            firstHeaders["Idempotency-Key"]
        );
    });

    it("submits only reviewed receipt values through the confirmed item contract", async () => {
        render(
            <CreateExpenseProvider>
                <CreateExpenseHarness />
            </CreateExpenseProvider>
        );
        await waitFor(() => expect(screen.getByTestId("members")).toHaveTextContent("1"));

        fireEvent.click(screen.getByRole("button", { name: "Apply receipt" }));
        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));

        await waitFor(() => expect(apiFetchMock.mock.calls.some(([path]) => path === "/create_expense")).toBe(true));
        const request = apiFetchMock.mock.calls.find(([path]) => path === "/create_expense")?.[1] as RequestInit;
        expect(JSON.parse(request.body as string)).toMatchObject({
            description: "Cafe",
            providerName: "Cafe",
            occurredOn: "2026-09-27",
            currency: "CAD",
            subTotal: "5.00",
            taxFeeTip: "1.65",
            total: "6.65",
            items: [{
                description: "Coffee", quantity: "2", unit: "cup",
                unitPrice: "2.50", lineTotal: "5.00",
            }],
        });
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
