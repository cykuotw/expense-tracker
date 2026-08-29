import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import AccountSettings from "./AccountSettings";

const {
    apiFetchMock,
    errorMessageMock,
    toastSuccessMock,
    googleConfigState,
} = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    errorMessageMock: vi.fn(),
    toastSuccessMock: vi.fn(),
    googleConfigState: { enabled: true },
}));

vi.mock("../configs/config", () => ({
    get GOOGLE_OAUTH_ENABLED() {
        return googleConfigState.enabled;
    },
}));

vi.mock("../components/auth/GoogleSignInButton", () => ({
    default: ({
        onCredentialResponse,
    }: {
        onCredentialResponse?: (
            response: GoogleCredentialResponse,
        ) => void | Promise<void>;
    }) => (
        <button
            type="button"
            onClick={() =>
                void onCredentialResponse?.({
                    credential: "google-id-token",
                    select_by: "ui-button",
                })
            }
        >
            Choose Google account
        </button>
    ),
}));

vi.mock("../lib/api", () => ({
    apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    getResponseErrorMessage: (...args: unknown[]) => errorMessageMock(...args),
}));

vi.mock("react-hot-toast", () => ({
    toast: { success: toastSuccessMock },
}));

const localAccount = {
    firstname: "Local",
    lastname: "User",
    nickname: "LU",
    email: "local@example.com",
    googleConnected: false,
    passwordChangeAllowed: true,
};

function response(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

describe("AccountSettings", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        googleConfigState.enabled = true;
        errorMessageMock.mockResolvedValue("Request failed");
        apiFetchMock.mockImplementation((path: string, init?: RequestInit) => {
            if (path === "/account/google/link") {
                return Promise.resolve(
                    response({ ...localAccount, googleConnected: true }),
                );
            }
            if (path === "/account" && init?.method === "PATCH") {
                const body = JSON.parse(String(init.body));
                return Promise.resolve(response({ ...localAccount, ...body }));
            }
            return Promise.resolve(response(localAccount));
        });
    });

    afterEach(cleanup);

    it("updates editable profile fields while keeping email read-only", async () => {
        render(<AccountSettings />);

        const email = await screen.findByRole("textbox", { name: "Email" });
        const saveProfile = screen.getByRole("button", { name: "Save profile" });
        expect(email).toHaveValue("local@example.com");
        expect(email).toHaveAttribute("readonly");
        expect(saveProfile).toBeDisabled();

        fireEvent.change(screen.getByRole("textbox", { name: "First name" }), {
            target: { value: " Taylor " },
        });
        expect(saveProfile).toBeEnabled();
        fireEvent.change(screen.getByRole("textbox", { name: "Last name" }), {
            target: { value: " Swift " },
        });
        fireEvent.click(saveProfile);

        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith("/account", {
                method: "PATCH",
                body: JSON.stringify({
                    firstname: "Taylor",
                    lastname: "Swift",
                    nickname: "LU",
                }),
            }),
        );
        expect(toastSuccessMock).toHaveBeenCalledWith("Profile updated");
        expect(saveProfile).toBeDisabled();
    });

    it("disables profile saving again when changes are reverted", async () => {
        render(<AccountSettings />);
        const firstName = await screen.findByRole("textbox", { name: "First name" });
        const saveProfile = screen.getByRole("button", { name: "Save profile" });

        fireEvent.change(firstName, { target: { value: "Changed" } });
        expect(saveProfile).toBeEnabled();

        fireEvent.change(firstName, { target: { value: "Local" } });
        expect(saveProfile).toBeDisabled();
    });

    it("validates and changes a local account password", async () => {
        render(<AccountSettings />);
        await screen.findByDisplayValue("local@example.com");

        fireEvent.change(screen.getByLabelText("Current password"), {
            target: { value: "old-password" },
        });
        fireEvent.change(screen.getByLabelText("New password"), {
            target: { value: "new-password" },
        });
        fireEvent.change(screen.getByLabelText("Confirm new password"), {
            target: { value: "new-password" },
        });
        const changePassword = screen.getByRole("button", {
            name: "Change password",
        });
        expect(changePassword).toBeDisabled();
        await waitFor(() => expect(changePassword).toBeEnabled());
        fireEvent.click(changePassword);

        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith("/account/password", {
                method: "PATCH",
                body: JSON.stringify({
                    currentPassword: "old-password",
                    newPassword: "new-password",
                }),
            }),
        );
        expect(toastSuccessMock).toHaveBeenCalledWith(
            "Password changed. Other sessions were signed out.",
        );
    });

    it("shows each password independently and applies the new-password minimum length", async () => {
        render(<AccountSettings />);
        await screen.findByDisplayValue("local@example.com");

        const currentPassword = screen.getByLabelText("Current password");
        const newPassword = screen.getByLabelText("New password");
        const confirmPassword = screen.getByLabelText("Confirm new password");

        expect(currentPassword).toHaveAttribute("type", "password");
        expect(currentPassword).not.toHaveAttribute("minlength");
        expect(newPassword).toHaveAttribute("minlength", "8");
        expect(confirmPassword).toHaveAttribute("minlength", "8");

        fireEvent.click(
            screen.getByRole("button", { name: "Show current password" }),
        );
        expect(currentPassword).toHaveAttribute("type", "text");
        expect(newPassword).toHaveAttribute("type", "password");
        expect(confirmPassword).toHaveAttribute("type", "password");

        fireEvent.click(
            screen.getByRole("button", { name: "Hide current password" }),
        );
        expect(currentPassword).toHaveAttribute("type", "password");
    });

    it("shows debounced password length and confirmation feedback", async () => {
        render(<AccountSettings />);
        await screen.findByDisplayValue("local@example.com");

        const newPassword = screen.getByLabelText("New password");
        const confirmPassword = screen.getByLabelText("Confirm new password");

        fireEvent.change(newPassword, { target: { value: "short" } });
        expect(
            screen.queryByText("Password must be at least 8 characters."),
        ).not.toBeInTheDocument();
        expect(
            await screen.findByText("Password must be at least 8 characters."),
        ).toHaveClass("text-destructive");

        fireEvent.change(newPassword, { target: { value: "valid-password" } });
        expect(
            await screen.findByText("Password length is valid."),
        ).toHaveClass("text-success");

        fireEvent.change(confirmPassword, { target: { value: "different" } });
        expect(await screen.findByText("Passwords do not match.")).toHaveClass(
            "text-destructive",
        );

        fireEvent.change(confirmPassword, {
            target: { value: "valid-password" },
        });
        expect(await screen.findByText("Passwords match.")).toHaveClass(
            "text-success",
        );
    });

    it("keeps password submission disabled until every check passes", async () => {
        render(<AccountSettings />);
        await screen.findByDisplayValue("local@example.com");

        const currentPassword = screen.getByLabelText("Current password");
        const newPassword = screen.getByLabelText("New password");
        const confirmPassword = screen.getByLabelText("Confirm new password");
        const changePassword = screen.getByRole("button", {
            name: "Change password",
        });

        expect(changePassword).toBeDisabled();
        fireEvent.change(currentPassword, {
            target: { value: "same-password" },
        });
        fireEvent.change(newPassword, { target: { value: "same-password" } });
        fireEvent.change(confirmPassword, {
            target: { value: "same-password" },
        });

        expect(
            await screen.findByText(
                "New password must be different from your current password.",
            ),
        ).toHaveClass("text-destructive");
        expect(changePassword).toBeDisabled();

        fireEvent.change(newPassword, { target: { value: "new-password" } });
        fireEvent.change(confirmPassword, {
            target: { value: "new-password" },
        });
        expect(changePassword).toBeDisabled();

        expect(
            await screen.findByText(
                "New password is different from your current password.",
            ),
        ).toHaveClass("text-success");
        expect(await screen.findByText("Password length is valid.")).toHaveClass(
            "text-success",
        );
        expect(await screen.findByText("Passwords match.")).toHaveClass(
            "text-success",
        );
        expect(changePassword).toBeEnabled();

        fireEvent.change(confirmPassword, { target: { value: "not-matching" } });
        expect(changePassword).toBeDisabled();
        expect(await screen.findByText("Passwords do not match.")).toHaveClass(
            "text-destructive",
        );
        expect(changePassword).toBeDisabled();
    });

    it("shows Google-managed state without a password form", async () => {
        apiFetchMock.mockResolvedValue(
            response({
                ...localAccount,
                googleConnected: true,
                passwordChangeAllowed: false,
            }),
        );
        render(<AccountSettings />);

        expect(await screen.findByText("Managed by Google")).toBeInTheDocument();
        expect(
            screen.queryByRole("button", { name: "Change password" }),
        ).not.toBeInTheDocument();
        expect(screen.getByText("Connected")).toBeInTheDocument();
        expect(
            screen.queryByRole("button", { name: "Connect Google" }),
        ).not.toBeInTheDocument();
    });

    it("explicitly links Google after confirming the current password", async () => {
        render(<AccountSettings />);

        const connect = await screen.findByRole("button", {
            name: "Connect Google",
        });
        fireEvent.click(connect);

        const continueButton = screen.getByRole("button", {
            name: "Continue with Google",
        });
        expect(continueButton).toBeDisabled();

        const confirmation = screen.getByLabelText("Confirm current password");
        fireEvent.change(confirmation, {
            target: { value: "current-password" },
        });
        fireEvent.click(
            screen.getByRole("button", { name: "Choose Google account" }),
        );

        await waitFor(() =>
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/account/google/link",
                {
                    method: "POST",
                    headers: { Authorization: "Bearer google-id-token" },
                    body: JSON.stringify({
                        currentPassword: "current-password",
                    }),
                },
                { authMode: "none" },
            ),
        );
        expect(toastSuccessMock).toHaveBeenCalledWith(
            "Google account connected. Other sessions were signed out.",
        );
        expect(screen.getByText("Connected")).toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "Change password" }),
        ).toBeInTheDocument();
    });

    it("shows a recoverable Google linking error without losing the form", async () => {
        errorMessageMock.mockResolvedValueOnce(
            "Use the Google account that matches this account email.",
        );
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/account/google/link") {
                return Promise.resolve(response({}, 409));
            }
            return Promise.resolve(response(localAccount));
        });
        render(<AccountSettings />);

        fireEvent.click(
            await screen.findByRole("button", { name: "Connect Google" }),
        );
        fireEvent.change(screen.getByLabelText("Confirm current password"), {
            target: { value: "current-password" },
        });
        fireEvent.click(
            screen.getByRole("button", { name: "Choose Google account" }),
        );

        expect(
            await screen.findByText(
                "Use the Google account that matches this account email.",
            ),
        ).toHaveClass("text-destructive");
        expect(
            screen.getByLabelText("Confirm current password"),
        ).toHaveValue("current-password");
    });

    it("does not offer linking when Google OAuth is disabled", async () => {
        googleConfigState.enabled = false;
        render(<AccountSettings />);

        expect(
            await screen.findByText(
                "Google linking is unavailable for this account.",
            ),
        ).toBeInTheDocument();
        expect(
            screen.queryByRole("button", { name: "Connect Google" }),
        ).not.toBeInTheDocument();
    });

    it("shows a recoverable loading error", async () => {
        apiFetchMock.mockResolvedValue(response({}, 500));
        render(<AccountSettings />);

        expect(await screen.findByRole("alert")).toHaveTextContent("Request failed");
        expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    });
});
