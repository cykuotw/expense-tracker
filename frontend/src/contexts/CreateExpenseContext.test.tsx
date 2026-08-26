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
            <output data-testid="members">{context.ledgers.length}</output>
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

        fireEvent.submit(screen.getByRole("form", { name: "expense form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Failed to create expense."
            );
        });
        expect(screen.getByTestId("indicator")).toHaveTextContent("idle");
        expect(toastSuccessMock).not.toHaveBeenCalled();
        expect(navigateMock).not.toHaveBeenCalled();
    });
});
