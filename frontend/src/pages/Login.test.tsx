import type { ReactNode } from "react";
import {
    cleanup,
    fireEvent,
    render,
    screen,
    waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Login from "./Login";

const {
    navigateMock,
    markLoggedInMock,
    apiFetchMock,
    getResponseErrorMessageMock,
    toastErrorMock,
    configState,
} = vi.hoisted(() => ({
    navigateMock: vi.fn(),
    markLoggedInMock: vi.fn(),
    apiFetchMock: vi.fn(),
    getResponseErrorMessageMock: vi.fn(),
    toastErrorMock: vi.fn(),
    configState: {
        googleOAuthEnabled: true,
    },
}));

vi.mock("../configs/config", () => ({
    get GOOGLE_OAUTH_ENABLED() {
        return configState.googleOAuthEnabled;
    },
}));

vi.mock("react-router-dom", () => ({
    useNavigate: () => navigateMock,
}));

vi.mock("../contexts/LoginContext", () => ({
    LoginProvider: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("../components/auth/LoginForm", () => ({
    default: () => <div>login form</div>,
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
                    credential: "test-google-id-token",
                    select_by: "btn",
                })
            }
        >
            Google Sign In
        </button>
    ),
}));

vi.mock("../hooks/AuthContextHooks", () => ({
    useAuth: () => ({
        markLoggedIn: markLoggedInMock,
    }),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseErrorMessage: (...args: unknown[]) =>
        getResponseErrorMessageMock(...args),
}));

vi.mock("react-hot-toast", () => ({
    toast: {
        error: toastErrorMock,
    },
}));

describe("Login", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        configState.googleOAuthEnabled = true;
    });

    afterEach(() => {
        cleanup();
    });

    it("exchanges the Google credential and completes login", async () => {
        apiFetchMock.mockResolvedValue({ ok: true } satisfies Partial<Response>);
        markLoggedInMock.mockResolvedValue(true);

        render(<Login />);

        fireEvent.click(screen.getByRole("button", { name: "Google Sign In" }));

        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/auth/google/exchange",
                {
                    method: "POST",
                    headers: {
                        Authorization: "Bearer test-google-id-token",
                    },
                },
                { authMode: "none" },
            );
        });

        await waitFor(() => {
            expect(markLoggedInMock).toHaveBeenCalled();
            expect(navigateMock).toHaveBeenCalledWith("/", { replace: true });
        });
    });

    it("shows the backend error when exchange fails", async () => {
        apiFetchMock.mockResolvedValue({ ok: false } satisfies Partial<Response>);
        getResponseErrorMessageMock.mockResolvedValue("Google sign-in failed");

        render(<Login />);

        fireEvent.click(screen.getByRole("button", { name: "Google Sign In" }));

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith(
                "Google sign-in failed",
            );
        });
    });

    it("hides Google sign-in when Google OAuth is disabled", () => {
        configState.googleOAuthEnabled = false;

        render(<Login />);

        expect(
            screen.queryByRole("button", { name: "Google Sign In" }),
        ).not.toBeInTheDocument();
        expect(screen.getByText("login form")).toBeInTheDocument();
    });
});
