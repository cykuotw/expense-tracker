import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import GoogleSignInButton from "./GoogleSignInButton";

const { toastError } = vi.hoisted(() => ({
    toastError: vi.fn(),
}));

vi.mock("../../configs/config", () => ({
    GOOGLE_CLIENT_ID: "test-google-client-id",
}));

vi.mock("react-hot-toast", () => ({
    toast: {
        error: toastError,
    },
}));

vi.mock("../../lib/googleIdentity", () => ({
    loadGoogleAccountsId: vi.fn(),
}));

import { loadGoogleAccountsId } from "../../lib/googleIdentity";

describe("GoogleSignInButton", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("initializes GIS and renders the Google button", async () => {
        const initialize = vi.fn();
        const renderButton = vi.fn();

        vi.mocked(loadGoogleAccountsId).mockResolvedValue({
            initialize,
            renderButton,
        });

        render(<GoogleSignInButton />);

        await waitFor(() => {
            expect(initialize).toHaveBeenCalledWith({
                client_id: "test-google-client-id",
                callback: expect.any(Function),
                ux_mode: "popup",
            });
        });

        expect(renderButton).toHaveBeenCalledWith(
            expect.any(HTMLDivElement),
            expect.objectContaining({
                type: "standard",
                theme: "outline",
                shape: "pill",
            })
        );

        await waitFor(() => {
            expect(
                screen.queryByText("Loading Google sign-in...")
            ).not.toBeInTheDocument();
        });
    });

    it("shows a user-visible error when GIS loading fails", async () => {
        vi.mocked(loadGoogleAccountsId).mockRejectedValue(
            new Error("SDK unavailable"),
        );

        render(<GoogleSignInButton />);

        await waitFor(() => {
            expect(toastError).toHaveBeenCalledWith("SDK unavailable");
        });

        expect(
            screen.queryByText("Loading Google sign-in..."),
        ).not.toBeInTheDocument();
    });
});
