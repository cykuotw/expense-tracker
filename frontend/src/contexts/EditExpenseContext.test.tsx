import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { EditExpenseProvider } from "./EditExpenseContext";
import { useEditExpense } from "../hooks/EditExpenseContextHooks";

const { apiFetchMock, navigateMock, toastErrorMock, toastSuccessMock } =
    vi.hoisted(() => ({
        apiFetchMock: vi.fn(),
        navigateMock: vi.fn(),
        toastErrorMock: vi.fn(),
        toastSuccessMock: vi.fn(),
    }));

vi.mock("react-router-dom", () => ({
    useNavigate: () => navigateMock,
    useParams: () => ({ id: "expense-1" }),
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
    splitRule: "Equally",
    items: [],
    ledgers: [
        {
            id: "ledger-1",
            lenderUserId: "user-1",
            lenderUsername: "Current",
            borrowerUserId: "user-1",
            borrowerUsername: "Current",
            share: "10.00",
        },
    ],
};

function EditExpenseHarness() {
    const context = useEditExpense();

    return (
        <form aria-label="edit form" onSubmit={context.handleUpdateExpense}>
            <output data-testid="currency">{context.formData.currency}</output>
            <output data-testid="indicator">
                {context.indicatorShow ? "loading" : "idle"}
            </output>
            <output data-testid="has-changes">
                {context.hasChanges ? "changed" : "unchanged"}
            </output>
            <button
                type="button"
                onClick={() =>
                    context.setFormData((current) => ({
                        ...current,
                        description: "Updated dinner",
                    }))
                }
            >
                Change description
            </button>
            <button
                type="button"
                onClick={() =>
                    context.setFormData((current) => ({
                        ...current,
                        description: "Dinner",
                    }))
                }
            >
                Restore description
            </button>
            <button type="submit">Update</button>
        </form>
    );
}

describe("EditExpenseProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/groups") {
                    return Promise.resolve(
                        jsonResponse([
                            {
                                id: "group-1",
                                groupName: "Group",
                                description: "",
                                currency: "CAD",
                            },
                        ])
                    );
                }
                if (path === "/expense_types") {
                    return Promise.resolve(jsonResponse([]));
                }
                if (path === "/expense/expense-1" && init.method === "PUT") {
                    return Promise.resolve(
                        jsonResponse({ error: "user not permitted" }, 403)
                    );
                }
                if (path === "/expense/expense-1") {
                    return Promise.resolve(jsonResponse(expenseDetail));
                }
                if (path === "/group_member/group-1") {
                    return Promise.resolve(
                        jsonResponse([
                            { userId: "user-1", username: "Current" },
                        ])
                    );
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );
    });

    afterEach(() => {
        cleanup();
    });

    it("shows the parsed update error without success behavior", async () => {
        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("currency")).toHaveTextContent("CAD");
        });

        fireEvent.click(screen.getByRole("button", { name: "Change description" }));

        fireEvent.submit(screen.getByRole("form", { name: "edit form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith("user not permitted");
        });
		const updateRequest = apiFetchMock.mock.calls.find(
			([path, init]) =>
				path === "/expense/expense-1" && init?.method === "PUT"
		)?.[1] as RequestInit;
		expect(JSON.parse(updateRequest.body as string)).toMatchObject({
			occurredOn: "2026-01-01",
		});
        expect(screen.getByTestId("indicator")).toHaveTextContent("idle");
        expect(toastSuccessMock).not.toHaveBeenCalled();
        expect(navigateMock).not.toHaveBeenCalled();
    });

    it("tracks edits relative to the loaded expense", async () => {
        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("currency")).toHaveTextContent("CAD");
        });
        expect(screen.getByTestId("has-changes")).toHaveTextContent("unchanged");

        fireEvent.click(screen.getByRole("button", { name: "Change description" }));

        expect(screen.getByTestId("has-changes")).toHaveTextContent("changed");

        fireEvent.click(screen.getByRole("button", { name: "Restore description" }));

        expect(screen.getByTestId("has-changes")).toHaveTextContent("unchanged");
    });

    it("keeps the editor mounted when the expense payload is null", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/groups" || path === "/expense_types") {
                return Promise.resolve(jsonResponse([]));
            }
            if (path === "/expense/expense-1") {
                return Promise.resolve(jsonResponse(null));
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Failed to load expense."
            );
        });
        expect(screen.getByRole("form", { name: "edit form" })).toBeInTheDocument();
        expect(apiFetchMock).not.toHaveBeenCalledWith(
            "/group_member/undefined",
            expect.anything()
        );
    });
});
