import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AddMemberProvider } from "./AddMemberContext";
import { useAddMember } from "../hooks/AddMemberContextHooks";

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

vi.mock("../hooks/useDebounce", () => ({
    default: (value: string) => value,
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

function AddMemberHarness() {
    const context = useAddMember();

    return (
        <>
            <form aria-label="member form" onSubmit={context.handleSubmitRelatedUsers}>
                {context.relatedUserList.map((user) => (
                    <input
                        key={user.userId}
                        type="checkbox"
                        name="candidate[]"
                        value={user.userId}
                        defaultChecked={user.existInGroup}
                    />
                ))}
                <button type="submit">Update</button>
            </form>
            <input
                aria-label="email"
                value={context.email}
                onChange={(event) => context.setEmail(event.target.value)}
            />
            <output data-testid="new-member">{context.newMember?.id ?? ""}</output>
            <output data-testid="loading">
                {context.loading ? "loading" : "idle"}
            </output>
        </>
    );
}

function renderProvider() {
    return render(
        <AddMemberProvider>
            <AddMemberHarness />
        </AddMemberProvider>
    );
}

describe("AddMemberProvider error handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    afterEach(() => {
        cleanup();
    });

    it("shows a related-member backend error without success behavior", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/related_member")) {
                return Promise.resolve(
                    jsonResponse([
                        {
                            userId: "user-1",
                            username: "Friend",
                            existInGroup: true,
                        },
                    ])
                );
            }
            if (path === "/group_member") {
                return Promise.resolve(
                    jsonResponse(
                        {
                            error: "user not permitted",
                            code: "user_not_permitted",
                        },
                        403
                    )
                );
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        renderProvider();
        await waitFor(() => expect(screen.getAllByRole("checkbox")).toHaveLength(1));

        fireEvent.submit(screen.getByRole("form", { name: "member form" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith("user not permitted");
        });
        expect(toastSuccessMock).not.toHaveBeenCalled();
        expect(navigateMock).not.toHaveBeenCalled();
        expect(screen.getByTestId("loading")).toHaveTextContent("idle");
    });

    it("does not run user lookup when the email check fails", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/related_member")) {
                return Promise.resolve(jsonResponse([]));
            }
            if (path === "/checkEmail") {
                return Promise.resolve(
                    jsonResponse({ error: "email service unavailable" }, 503)
                );
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        renderProvider();
        fireEvent.change(screen.getByLabelText("email"), {
            target: { value: "person@example.com" },
        });

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "email service unavailable",
                { id: "email-validation" }
            );
        });
        expect(
            apiFetchMock.mock.calls.some(([path]) =>
                String(path).startsWith("/userInfo")
            )
        ).toBe(false);
        expect(screen.getByTestId("new-member")).toBeEmptyDOMElement();
    });

    it("does not apply an error response as user lookup data", async () => {
        apiFetchMock.mockImplementation((path: string) => {
            if (path.startsWith("/related_member")) {
                return Promise.resolve(jsonResponse([]));
            }
            if (path === "/checkEmail") {
                return Promise.resolve(jsonResponse({ exist: true }));
            }
            if (path.startsWith("/userInfo")) {
                return Promise.resolve(
                    jsonResponse({ error: "user lookup failed" }, 500)
                );
            }
            throw new Error(`Unexpected path: ${path}`);
        });

        renderProvider();
        fireEvent.change(screen.getByLabelText("email"), {
            target: { value: "person@example.com" },
        });

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith("user lookup failed", {
                id: "email-validation",
            });
        });
        expect(screen.getByTestId("new-member")).toBeEmptyDOMElement();
        expect(screen.getByTestId("loading")).toHaveTextContent("idle");
    });
});
