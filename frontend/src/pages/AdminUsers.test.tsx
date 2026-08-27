import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import AdminUsers from "./AdminUsers";

const {
    apiFetchMock,
    getResponseErrorMessageMock,
    toastErrorMock,
    toastSuccessMock,
    clipboardWriteMock,
} = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getResponseErrorMessageMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
    clipboardWriteMock: vi.fn(),
}));

vi.mock("../hooks/AuthContextHooks", () => ({
    useAuth: () => ({ userID: "admin-self" }),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseErrorMessage: (...args: unknown[]) =>
        getResponseErrorMessageMock(...args),
}));

vi.mock("react-hot-toast", () => ({
    toast: {
        error: toastErrorMock,
        success: toastSuccessMock,
    },
}));

const managementData = {
    users: [
        {
            id: "admin-self",
            firstname: "Current",
            lastname: "Admin",
            nickname: "",
            email: "admin@example.com",
            role: "admin",
            isActive: true,
            createTime: "2026-01-01T00:00:00Z",
        },
        {
            id: "user-1",
            firstname: "Regular",
            lastname: "User",
            nickname: "",
            email: "user@example.com",
            role: "user",
            isActive: true,
            createTime: "2026-01-02T00:00:00Z",
        },
    ],
    invitations: [
        {
            id: "invite-1",
            email: "invited@example.com",
            status: "invited",
            createdAt: "2026-01-03T00:00:00Z",
            expiresAt: "2026-12-31T00:00:00Z",
            usedAt: null,
        },
    ],
};

function response(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function renderPage() {
    return render(
        <MemoryRouter>
            <AdminUsers />
        </MemoryRouter>,
    );
}

describe("AdminUsers", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        Object.defineProperty(navigator, "clipboard", {
            configurable: true,
            value: { writeText: clipboardWriteMock },
        });
        clipboardWriteMock.mockResolvedValue(undefined);
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/admin/users") return Promise.resolve(response(managementData));
            if (path === "/admin/invitations/invite-1/link") {
                return Promise.resolve(response({ token: "invite-token" }));
            }
            return Promise.resolve(response({}));
        });
    });

    afterEach(() => {
        vi.restoreAllMocks();
        cleanup();
    });

    it("shows account and invitation statuses and locks self-management", async () => {
        renderPage();

        expect(await screen.findByText("user@example.com")).toBeInTheDocument();
        expect(screen.getByText("invited@example.com")).toBeInTheDocument();
        expect(screen.getByText("invited")).toBeInTheDocument();

        const selfCard = screen.getByText("admin@example.com").closest("article");
        expect(selfCard).not.toBeNull();
        expect(within(selfCard!).getByRole("combobox")).toBeDisabled();
        expect(
            within(selfCard!).getByRole("button", { name: "Disable account" }),
        ).toBeDisabled();
    });

    it("updates role and account status with separate API calls", async () => {
        renderPage();
        const userEmail = await screen.findByText("user@example.com");
        const userCard = userEmail.closest("article");
        expect(userCard).not.toBeNull();

        fireEvent.change(within(userCard!).getByRole("combobox"), {
            target: { value: "admin" },
        });
        const roleDialog = screen.getByRole("alertdialog", {
            name: "Change this user’s role?",
        });
        expect(roleDialog).toHaveTextContent(
            "user@example.com · Regular user → Administrator",
        );
        expect(apiFetchMock).not.toHaveBeenCalledWith(
            "/admin/users/user-1/role",
            expect.anything(),
        );
        fireEvent.click(
            within(roleDialog).getByRole("button", {
                name: "Change to Administrator",
            }),
        );
        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/admin/users/user-1/role",
                { method: "PATCH", body: JSON.stringify({ role: "admin" }) },
            ),
        );

        fireEvent.click(
            within(userCard!).getByRole("button", { name: "Disable account" }),
        );
        const statusDialog = screen.getByRole("alertdialog", {
            name: "Disable this account?",
        });
        fireEvent.click(
            within(statusDialog).getByRole("button", {
                name: "Disable account",
            }),
        );
        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/admin/users/user-1/status",
                {
                    method: "PATCH",
                    body: JSON.stringify({ isActive: false }),
                },
            ),
        );
    });

    it("cancels a role change without calling the mutation API", async () => {
        renderPage();
        const userCard = (await screen.findByText("user@example.com")).closest(
            "article",
        );
        expect(userCard).not.toBeNull();

        fireEvent.change(within(userCard!).getByRole("combobox"), {
            target: { value: "admin" },
        });
        fireEvent.click(
            within(screen.getByRole("alertdialog")).getByRole("button", {
                name: "Cancel",
            }),
        );

        expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
        expect(apiFetchMock).not.toHaveBeenCalledWith(
            "/admin/users/user-1/role",
            expect.anything(),
        );
        expect(within(userCard!).getByRole("combobox")).toHaveValue("user");
    });

    it.each([
        ["a generic invitation", "", ""],
        [
            "an email-bound invitation",
            " invited@example.com ",
            "invited@example.com",
        ],
    ])("creates %s from user management", async (_label, input, expected) => {
        renderPage();
        await screen.findByText("user@example.com");

        const emailInput = screen.getByRole("textbox", {
            name: "Email (optional)",
        });
        fireEvent.change(emailInput, { target: { value: input } });
        fireEvent.click(screen.getByRole("button", { name: "Create invite" }));

        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith("/invitations", {
                method: "POST",
                body: JSON.stringify({ email: expected }),
            }),
        );
        await waitFor(() => expect(emailInput).toHaveValue(""));
        expect(toastSuccessMock).toHaveBeenCalledWith("Invitation created");
    });

    it("confirms before expiring an invitation", async () => {
        renderPage();
        await screen.findByText("invited@example.com");

        fireEvent.click(screen.getByRole("button", { name: "Expire" }));
        const dialog = screen.getByRole("alertdialog", {
            name: "Expire this invitation?",
        });
        expect(dialog).toHaveTextContent("invited@example.com");
        fireEvent.click(
            within(dialog).getByRole("button", { name: "Expire invitation" }),
        );

        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/admin/invitations/invite-1/expire",
                { method: "POST" },
            ),
        );
    });

    it("retrieves an invitation token only when copying the link", async () => {
        renderPage();
        await screen.findByText("invited@example.com");

        fireEvent.click(screen.getByRole("button", { name: "Copy link" }));

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/admin/invitations/invite-1/link",
            );
            expect(clipboardWriteMock).toHaveBeenCalledWith(
                "http://localhost:3000/register?token=invite-token",
            );
        });
    });
});
