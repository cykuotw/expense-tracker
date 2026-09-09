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

function editOptions(
    expense: typeof expenseDetail | null = expenseDetail,
    members = [{ userId: "user-1", username: "Current" }]
) {
    return {
        expense,
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
        group: { currency: "CAD", members },
    };
}

function EditExpenseHarness() {
    const context = useEditExpense();

    return (
        <form aria-label="edit form" onSubmit={context.handleUpdateExpense}>
            <output data-testid="currency">{context.formData.currency}</output>
            <output data-testid="split-rule">{context.formData.splitRule}</output>
            <output data-testid="payer">{context.formData.payerUserId}</output>
            <output data-testid="members">{context.groupMembers.length}</output>
            <output data-testid="member-load-status">
                {context.groupMembersLoadStatus}
            </output>
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
            <button
                type="button"
                onClick={() =>
                    context.setFormData((current) => ({
                        ...current,
                        groupId: "group-2",
                    }))
                }
            >
                Select second group
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
                if (path === "/expense/expense-1" && init.method === "PUT") {
                    return Promise.resolve(
                        jsonResponse({ error: "user not permitted" }, 403)
                    );
                }
                if (path === "/expense/expense-1/edit-options") {
                    return Promise.resolve(jsonResponse(editOptions()));
                }
                if (
                    path ===
                    "/expense/expense-1/edit-options?groupId=group-2"
                ) {
                    return Promise.resolve(
                        jsonResponse(
                            editOptions(expenseDetail, [
                                { userId: "user-2", username: "Other" },
                                { userId: "user-1", username: "Current" },
                            ])
                        )
                    );
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );
    });

    afterEach(() => {
        cleanup();
    });

    it("loads all initial editor data with one page request", async () => {
        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("member-load-status")).toHaveTextContent(
                "ready"
            );
        });
        const initialReads = apiFetchMock.mock.calls.filter(
            ([, init]) => init?.method === "GET"
        );
        expect(initialReads).toHaveLength(1);
        expect(initialReads[0][0]).toBe("/expense/expense-1/edit-options");
    });

    it("loads a newly selected group's members through the same page endpoint", async () => {
        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("member-load-status")).toHaveTextContent(
                "ready"
            );
        });

        fireEvent.click(
            screen.getByRole("button", { name: "Select second group" })
        );

        await waitFor(() => {
            expect(screen.getByTestId("members")).toHaveTextContent("2");
        });
        expect(apiFetchMock).toHaveBeenCalledWith(
            "/expense/expense-1/edit-options?groupId=group-2",
            expect.objectContaining({ method: "GET" })
        );
    });

    it("shows the parsed update error without success behavior", async () => {
        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );
        await waitFor(() => {
            expect(screen.getByTestId("currency")).toHaveTextContent("CAD");
            expect(screen.getByTestId("member-load-status")).toHaveTextContent(
                "ready"
            );
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

    it("aborts stale reads after leaving the editor", async () => {
        let expenseSignal: AbortSignal | undefined;
        apiFetchMock.mockImplementation(
            (path: string, init: RequestInit = {}) => {
                if (path === "/expense/expense-1/edit-options") {
                    expenseSignal = init.signal as AbortSignal;
                    return new Promise(() => undefined);
                }
                throw new Error(`Unexpected path: ${path}`);
            }
        );

        const view = render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );
        await waitFor(() => expect(expenseSignal).toBeDefined());

        view.unmount();

        expect(expenseSignal?.aborted).toBe(true);
        expect(toastErrorMock).not.toHaveBeenCalled();
    });

    it("orients a two-person split rule to the member editing the expense", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense/expense-1/edit-options") {
                return Promise.resolve(
                    jsonResponse(editOptions({
                        ...expenseDetail,
                        currentUser: "user-b",
                        splitRule: "You-Half",
                        ledgers: [
                            {
                                id: "ledger-a",
                                lenderUserId: "user-a",
                                lenderUsername: "A",
                                borrowerUserId: "user-a",
                                borrowerUsername: "A",
                                share: "5.00",
                            },
                            {
                                id: "ledger-b",
                                lenderUserId: "user-a",
                                lenderUsername: "A",
                                borrowerUserId: "user-b",
                                borrowerUsername: "B",
                                share: "5.00",
                            },
                        ],
                    }, [
                        { userId: "user-a", username: "A" },
                        { userId: "user-b", username: "B" },
                    ]))
                );
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        render(
            <EditExpenseProvider>
                <EditExpenseHarness />
            </EditExpenseProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("member-load-status")).toHaveTextContent(
                "ready"
            );
        });
        expect(screen.getByTestId("payer")).toHaveTextContent("user-a");
        expect(screen.getByTestId("split-rule")).toHaveTextContent(
            "Other-Half"
        );
        expect(screen.getByTestId("has-changes")).toHaveTextContent(
            "unchanged"
        );
    });

    it("keeps the editor mounted when the expense payload is null", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/expense/expense-1/edit-options") {
                return Promise.resolve(jsonResponse(editOptions(null)));
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
            expect.stringContaining("undefined"),
            expect.anything()
        );
    });
});
