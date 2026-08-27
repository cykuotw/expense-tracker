import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Register from "./Register";

const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseErrorMessage: vi.fn(),
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
            </Routes>
        </MemoryRouter>,
    );
}

describe("Register invitation email", () => {
    beforeEach(() => {
        vi.clearAllMocks();
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
});
