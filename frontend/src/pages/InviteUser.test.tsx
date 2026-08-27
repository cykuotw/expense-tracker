import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import InviteUser from "./InviteUser";

const { apiFetchMock, toastErrorMock, toastSuccessMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseErrorMessage: vi.fn(),
}));

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

describe("InviteUser", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation(
            (_path: string, init?: RequestInit) =>
                Promise.resolve(
                    init?.method === "POST"
                        ? jsonResponse({ token: "invite-token" }, 201)
                        : jsonResponse([]),
                ),
        );
    });

    afterEach(() => {
        cleanup();
    });

    it.each([
        ["a generic invitation", "", ""],
        [
            "an email-bound invitation",
            " invited@example.com ",
            "invited@example.com",
        ],
    ])("creates %s", async (_label, inputEmail, expectedEmail) => {
        render(<InviteUser />);

        const emailInput = screen.getByRole("textbox", {
            name: "Email (optional)",
        });
        fireEvent.change(emailInput, { target: { value: inputEmail } });
        fireEvent.click(
            screen.getByRole("button", { name: "Generate Invite" }),
        );

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith("/invitations", {
                method: "POST",
                body: JSON.stringify({ email: expectedEmail }),
            });
        });
        await waitFor(() => expect(emailInput).toHaveValue(""));
        expect(toastSuccessMock).toHaveBeenCalledWith("Invitation created", {
            duration: 1000,
        });
    });
});
