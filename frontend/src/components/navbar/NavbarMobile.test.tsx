import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import NavbarMobile from "./NavbarMobile";

const { useAuthMock } = vi.hoisted(() => ({
    useAuthMock: vi.fn(),
}));

vi.mock("../../hooks/AuthContextHooks", () => ({
    useAuth: () => useAuthMock(),
}));

describe("NavbarMobile account actions", () => {
    afterEach(() => {
        cleanup();
    });

    beforeEach(() => {
        useAuthMock.mockReturnValue({
            role: "admin",
            logout: vi.fn(),
        });
    });

    it("shows user management in the account sheet for administrators", () => {
        render(
            <MemoryRouter>
                <NavbarMobile />
            </MemoryRouter>,
        );

        expect(screen.queryByRole("link", { name: "Users" })).not.toBeInTheDocument();
        expect(
            screen
                .getByRole("navigation", { name: "Primary" })
                .querySelectorAll(".app-shell__mobile-item"),
        ).toHaveLength(3);
        fireEvent.click(
            screen.getByRole("button", { name: "Open account actions" }),
        );

        const settings = screen.getByRole("link", { name: "Settings" });
        const userManagement = screen.getByRole("link", {
            name: "User Management",
        });
        expect(settings.compareDocumentPosition(userManagement)).toBe(
            Node.DOCUMENT_POSITION_FOLLOWING,
        );
        expect(userManagement).toHaveAttribute("href", "/admin/users");
        expect(userManagement.closest(".mt-6")).not.toBeNull();
    });

    it("does not show user management to regular members", () => {
        useAuthMock.mockReturnValue({
            role: "user",
            logout: vi.fn(),
        });

        render(
            <MemoryRouter>
                <NavbarMobile />
            </MemoryRouter>,
        );

        fireEvent.click(
            screen.getByRole("button", { name: "Open account actions" }),
        );
        expect(
            screen.queryByRole("link", {
                name: "User Management",
                hidden: true,
            }),
        ).not.toBeInTheDocument();
    });
});
