import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ExpenseDetailProvider } from "./ExpenseDetailContext";
import { useExpenseDetail } from "../hooks/ExpenseDetailContextHooks";

const { apiFetchMock, navigateMock, paramsMock, toastErrorMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    navigateMock: vi.fn(),
    paramsMock: vi.fn(),
    toastErrorMock: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
    useNavigate: () => navigateMock,
    useParams: () => paramsMock(),
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
    toast: { error: toastErrorMock },
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function deferred<T>() {
    let resolve: (value: T) => void;
    const promise = new Promise<T>((resolvePromise) => {
        resolve = resolvePromise;
    });

    return { promise, resolve: resolve! };
}

const expenseDetail = {
    expenseId: "expense-1",
    description: "Dinner",
    createdByUserID: "user-1",
    createdByUsername: "Current",
    expenseTypeId: "type-1",
    expenseType: "General",
    subTotal: "10.00",
    taxFeeTip: "0.00",
    total: "10.00",
    currency: "CAD",
    expenseTime: "2026-01-01T00:00:00Z",
    occurredOn: "2026-01-01",
    invoiceUrl: "",
    currentUser: "user-1",
    groupId: "group-1",
    allocation: { mode: "equal", participants: [] },
    items: [],
    ledgers: [],
};

function ExpenseDetailHarness() {
    const context = useExpenseDetail();

    if (!context.expenseDetail) {
        return <output data-testid="load-error">{context.errorMessage ?? ""}</output>;
    }

    return (
        <form aria-label="delete form" onSubmit={context.handleDeleteExpense}>
            <output data-testid="description">
                {context.expenseDetail.description}
            </output>
            <button type="submit">Delete</button>
        </form>
    );
}

describe("ExpenseDetailProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        paramsMock.mockReturnValue({ id: "expense-1" });
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1") {
                    return Promise.resolve(jsonResponse(expenseDetail));
                }
                if (
                    path === "/delete_expense/expense-1" &&
                    init.method === "PUT"
                ) {
                    return Promise.resolve(
                        jsonResponse({ error: "delete not permitted" }, 403)
                    );
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );
    });

    afterEach(() => {
        cleanup();
    });

    it("navigates to the owning group after deletion", async () => {
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1") {
                    return Promise.resolve(jsonResponse(expenseDetail));
                }
                if (
                    path === "/delete_expense/expense-1" &&
                    init.method === "PUT"
                ) {
                    return Promise.resolve(jsonResponse({}));
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );

        render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );
        await screen.findByRole("form", { name: "delete form" });

        fireEvent.submit(screen.getByRole("form", { name: "delete form" }));

        await waitFor(() => {
            expect(navigateMock).toHaveBeenCalledWith("/group/group-1");
        });
    });

    it("shows the delete error without redirecting", async () => {
        render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );
        await screen.findByRole("form", { name: "delete form" });

        fireEvent.submit(screen.getByRole("form", { name: "delete form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "delete not permitted"
            );
        });
        expect(navigateMock).not.toHaveBeenCalled();
    });

    it("exposes an expense-load failure for the page to render", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense/expense-1") {
                return Promise.resolve(jsonResponse({ error: "expense unavailable" }, 404));
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("load-error")).toHaveTextContent(
                "expense unavailable"
            );
        });
    });

    it("aborts a superseded route request and ignores its late response", async () => {
        const firstRequest = deferred<Response>();
        let firstSignal: AbortSignal | undefined;
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1") {
                    firstSignal = init.signal as AbortSignal;
                    return firstRequest.promise;
                }
                if (path === "/expense/expense-2") {
                    return Promise.resolve(
                        jsonResponse({
                            ...expenseDetail,
                            expenseId: "expense-2",
                            description: "Latest dinner",
                        })
                    );
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );

        const view = render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );
        await waitFor(() => expect(firstSignal).toBeDefined());

        paramsMock.mockReturnValue({ id: "expense-2" });
        view.rerender(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );

        expect(firstSignal?.aborted).toBe(true);
        await waitFor(() =>
            expect(screen.getByTestId("description")).toHaveTextContent(
                "Latest dinner"
            )
        );

        await act(async () => {
            firstRequest.resolve(jsonResponse(expenseDetail));
        });
        expect(screen.getByTestId("description")).toHaveTextContent(
            "Latest dinner"
        );
    });

    it("aborts the expense request after leaving without showing an error", async () => {
        let expenseSignal: AbortSignal | undefined;
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1") {
                    expenseSignal = init.signal as AbortSignal;
                    return new Promise(() => undefined);
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );

        const view = render(
            <ExpenseDetailProvider>
                <ExpenseDetailHarness />
            </ExpenseDetailProvider>
        );
        await waitFor(() => expect(expenseSignal).toBeDefined());

        view.unmount();

        expect(expenseSignal?.aborted).toBe(true);
        expect(toastErrorMock).not.toHaveBeenCalled();
    });
});
