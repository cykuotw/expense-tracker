import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { UserRole, USER_ROLES } from "../../types/role";
import RouteGuard, { RouteGuardMode } from "./RouteGuard";

const { useAuthMock } = vi.hoisted(() => ({
    useAuthMock: vi.fn(),
}));

vi.mock("../../hooks/AuthContextHooks", () => ({
    useAuth: () => useAuthMock(),
}));

function renderGuard(
    mode: RouteGuardMode,
    isAuthenticated: boolean,
    role: UserRole | null,
) {
    useAuthMock.mockReturnValue({ isAuthenticated, role });
    const router = createMemoryRouter(
        [
            { path: "/", element: <p>Home</p> },
            { path: "/login", element: <p>Login</p> },
            {
                element: <RouteGuard mode={mode} />,
                children: [
                    { path: "/target", element: <p>Guarded content</p> },
                ],
            },
        ],
        { initialEntries: ["/previous", "/target"], initialIndex: 1 },
    );

    render(<RouterProvider router={router} />);
    return router;
}

describe("RouteGuard", () => {
    afterEach(() => {
        cleanup();
        vi.clearAllMocks();
    });

    it.each<[
        name: string,
        mode: RouteGuardMode,
        isAuthenticated: boolean,
        role: UserRole | null,
    ]>([
        ["allows a guest into a guest route", "guest", false, null],
        [
            "allows an authenticated user into an authenticated route",
            "authenticated",
            true,
            USER_ROLES.user,
        ],
        [
            "allows an administrator into an admin route",
            "admin",
            true,
            USER_ROLES.admin,
        ],
    ])("%s", async (_name, mode, isAuthenticated, role) => {
        const router = renderGuard(mode, isAuthenticated, role);

        expect(await screen.findByText("Guarded content")).toBeInTheDocument();
        expect(router.state.location.pathname).toBe("/target");
    });

    it.each<[
        name: string,
        mode: RouteGuardMode,
        isAuthenticated: boolean,
        role: UserRole | null,
        destination: string,
        destinationText: string,
    ]>([
        [
            "redirects an authenticated user away from a guest route",
            "guest",
            true,
            USER_ROLES.user,
            "/",
            "Home",
        ],
        [
            "redirects a guest away from an authenticated route",
            "authenticated",
            false,
            null,
            "/login",
            "Login",
        ],
        [
            "redirects a guest away from an admin route",
            "admin",
            false,
            null,
            "/login",
            "Login",
        ],
        [
            "redirects a non-admin away from an admin route",
            "admin",
            true,
            USER_ROLES.user,
            "/",
            "Home",
        ],
        [
            "redirects an authenticated user with no role away from an admin route",
            "admin",
            true,
            null,
            "/",
            "Home",
        ],
    ])(
        "%s with replacement navigation",
        async (
            _name,
            mode,
            isAuthenticated,
            role,
            destination,
            destinationText,
        ) => {
            const router = renderGuard(mode, isAuthenticated, role);

            await waitFor(() =>
                expect(router.state.location.pathname).toBe(destination),
            );
            expect(screen.getByText(destinationText)).toBeInTheDocument();
            expect(router.state.historyAction).toBe("REPLACE");
        },
    );
});
