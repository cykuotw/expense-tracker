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
            if (path === "/groups") return Promise.resolve(jsonResponse([]));
            if (path === "/expense_types") {
                return Promise.resolve(
                    jsonResponse([
                        { id: "type-1", category: "Other", name: "General" },
                    ])
                );
            }
            if (path === "/group_member/group-1") {
                return Promise.resolve(
                    jsonResponse([{ userId: "user-1", username: "Current" }])
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
        expect(screen.getByTestId("total")).toHaveTextContent("12.5");

        fireEvent.change(amountInput, { target: { value: "" } });
        expect(amountInput).toHaveValue(null);
        expect(screen.getByTestId("total")).toHaveTextContent("0");
    });
});
