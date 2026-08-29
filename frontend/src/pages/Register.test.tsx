import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Register from "./Register";

const { apiFetchMock, getResponseErrorMock, toastSuccessMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getResponseErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseError: (...args: unknown[]) => getResponseErrorMock(...args),
}));

vi.mock("../configs/config", () => ({
    GOOGLE_OAUTH_ENABLED: true,
}));

vi.mock("react-hot-toast", () => ({
    toast: { success: toastSuccessMock },
}));

vi.mock("../components/auth/GoogleSignInButton", () => ({
    default: ({
        onCredentialResponse,
    }: {
        onCredentialResponse?: (response: GoogleCredentialResponse) => void;
    }) => (
        <button
            type="button"
            onClick={() =>
                onCredentialResponse?.({
                    credential: "google-id-token",
                    select_by: "ui-button",
                })
            }
        >
            Register with Google
        </button>
    ),
}));

function invitationResponse(email: string) {
    return new Response(JSON.stringify({ valid: true, email }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    });
}

function renderRegister() {
    return render(
        <MemoryRouter initialEntries={["/register?token=invite-token"]}>
            <Routes>
                <Route path="/register" element={<Register />} />
                <Route path="/login" element={<div>Login destination</div>} />
            </Routes>
        </MemoryRouter>,
    );
}

describe("Register invitation email", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        window.history.replaceState({}, "", "/");
    });

    afterEach(() => {
        cleanup();
    });

    it("allows an email to be entered for a generic invitation", async () => {
        apiFetchMock.mockResolvedValue(invitationResponse(""));
        renderRegister();

        const emailInput = await screen.findByRole("textbox", { name: "Email" });
        expect(emailInput).not.toHaveAttribute("readonly");

        fireEvent.change(emailInput, {
            target: { value: "new-user@example.com" },
        });

        expect(emailInput).toHaveValue("new-user@example.com");
        expect(emailInput).not.toHaveAttribute("readonly");
        expect(
            screen.getByText("Enter the email address for your new account."),
        ).toBeInTheDocument();
    });

    it("locks the email supplied by an email-bound invitation", async () => {
        apiFetchMock.mockResolvedValue(
            invitationResponse("invited@example.com"),
        );
        renderRegister();

        const emailInput = await screen.findByRole("textbox", { name: "Email" });
        await waitFor(() =>
            expect(emailInput).toHaveValue("invited@example.com"),
        );
        expect(emailInput).toHaveAttribute("readonly");
        expect(
            screen.getByText(
                "This invitation is linked to this email address.",
            ),
        ).toBeInTheDocument();
    });

    it("registers a Google account with the captured invitation and navigates to login", async () => {
        apiFetchMock
            .mockResolvedValueOnce(invitationResponse(""))
            .mockResolvedValueOnce({ ok: true } satisfies Partial<Response>);
        renderRegister();

        fireEvent.click(
            await screen.findByRole("button", { name: "Register with Google" }),
        );

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenNthCalledWith(
                2,
                "/auth/google/register",
                {
                    method: "POST",
                    headers: { Authorization: "Bearer google-id-token" },
                    body: JSON.stringify({ token: "invite-token" }),
                },
                { authMode: "none" },
            );
        });
        expect(await screen.findByText("Login destination")).toBeInTheDocument();
        expect(toastSuccessMock).toHaveBeenCalledWith(
            "Google account registered. Continue with Google to log in.",
        );
    });

    it("shows a recovery message for an invitation email mismatch", async () => {
        apiFetchMock
            .mockResolvedValueOnce(invitationResponse("invited@example.com"))
            .mockResolvedValueOnce({ ok: false } satisfies Partial<Response>);
        getResponseErrorMock.mockResolvedValue({
            message: "mismatch",
            code: "INVITATION_EMAIL_MISMATCH",
        });
        renderRegister();

        fireEvent.click(
            await screen.findByRole("button", { name: "Register with Google" }),
        );

        expect(
            await screen.findByRole("alert"),
        ).toHaveTextContent(
            "Choose the Google account that matches the email address on this invitation.",
        );
    });

    it("removes the invitation token from the visible URL after capture", async () => {
        apiFetchMock.mockResolvedValue(invitationResponse(""));
        renderRegister();

        await screen.findByRole("textbox", { name: "Email" });
        await waitFor(() => expect(window.location.search).toBe(""));
        expect(window.location.pathname).toBe("/register");
    });
});
