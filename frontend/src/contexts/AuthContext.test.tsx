import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "./AuthContext";
import { useAuth } from "../hooks/AuthContextHooks";

const { apiFetchMock, setApiAuthFailureHandlerMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    setApiAuthFailureHandlerMock: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
    useLocation: () => ({ pathname: "/" }),
    useNavigate: () => vi.fn(),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    setApiAuthFailureHandler: (...args: unknown[]) =>
        setApiAuthFailureHandlerMock(...args),
}));

function AuthHarness() {
    const { isAuthenticated, isOffline, loading } = useAuth();

    return (
        <output data-testid="auth-state">
            {`${loading}:${isAuthenticated}:${isOffline}`}
        </output>
    );
}

describe("AuthProvider connectivity handling", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        Object.defineProperty(window.navigator, "onLine", {
            configurable: true,
            value: true,
        });
    });

    afterEach(() => {
        cleanup();
    });

    it("keeps a failed session check distinct from an unauthenticated response", async () => {
        apiFetchMock.mockRejectedValueOnce(new TypeError("network unavailable"));

        render(
            <AuthProvider>
                <AuthHarness />
            </AuthProvider>
        );

        await waitFor(() => {
            expect(screen.getByTestId("auth-state")).toHaveTextContent(
                "false:false:true"
            );
        });
    });
});
