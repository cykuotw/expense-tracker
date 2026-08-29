import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import Navbar from "./Navbar";

const { useAuthMock } = vi.hoisted(() => ({
    useAuthMock: vi.fn(),
}));

vi.mock("../../hooks/AuthContextHooks", () => ({
    useAuth: () => useAuthMock(),
}));

describe("Navbar account actions", () => {
    afterEach(() => {
        cleanup();
    });

    beforeEach(() => {
        useAuthMock.mockReturnValue({
            role: "admin",
            logout: vi.fn(),
        });
    });

    it("nests user management next to settings for administrators", () => {
        render(
            <MemoryRouter>
                <Navbar />
            </MemoryRouter>,
        );

        fireEvent.click(screen.getByRole("button", { name: "Account" }));

        const settings = screen.getByRole("link", { name: "Settings" });
        const userManagement = screen.getByRole("link", {
            name: "User Management",
        });
        expect(settings.compareDocumentPosition(userManagement)).toBe(
            Node.DOCUMENT_POSITION_FOLLOWING,
        );
        expect(userManagement).toHaveAttribute("href", "/admin/users");
        expect(userManagement.closest("ul")).toHaveClass("dropdown-content");
    });

    it("does not show user management to regular members", () => {
        useAuthMock.mockReturnValue({
            role: "user",
            logout: vi.fn(),
        });

        render(
            <MemoryRouter>
                <Navbar />
            </MemoryRouter>,
        );

        fireEvent.click(screen.getByRole("button", { name: "Account" }));
        expect(
            screen.queryByRole("link", {
                name: "User Management",
                hidden: true,
            }),
        ).not.toBeInTheDocument();
    });
});
