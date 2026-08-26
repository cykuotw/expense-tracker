import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GroupDetailProvider } from "./GroupDetailContext";
import { useGroupDetail } from "../hooks/GroupDetailContextHooks";

const { apiFetchMock, toastErrorMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    toastErrorMock: vi.fn(),
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
    toast: { error: toastErrorMock },
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
            <button type="button" onClick={context.handleSettle}>
                Settle
            </button>
        </>
    );
}

describe("GroupDetailProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/group/group-1") {
                return Promise.resolve(
                    jsonResponse({
                        groupName: "Group",
                        description: "",
                        currency: "CAD",
                        members: [],
                    })
                );
            }
            if (path === "/balance/group-1") {
                return Promise.resolve(
                    jsonResponse({
                        currency: "CAD",
                        currentUser: "user-1",
                        balances: [],
                    })
                );
            }
            if (path === "/expense_list/group-1/0") {
                return Promise.resolve(jsonResponse([]));
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

        fireEvent.click(screen.getByRole("button", { name: "Settle" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "settlement not permitted"
            );
        });
        expect(successLog).not.toHaveBeenCalledWith("Settlement successful");
        successLog.mockRestore();
    });
});
