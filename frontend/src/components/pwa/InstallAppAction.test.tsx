import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { PWAInstallProvider } from "../../contexts/PWAInstallProvider";
import InstallAppAction from "./InstallAppAction";

const matchMedia = vi.fn(() => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
}));

describe("InstallAppAction", () => {
    beforeEach(() => {
        vi.stubGlobal("matchMedia", matchMedia);
        matchMedia.mockClear();
    });

    afterEach(() => {
        cleanup();
        Reflect.deleteProperty(navigator, "userAgent");
        vi.unstubAllGlobals();
    });

    it("keeps the browser prompt until Account actions mount", async () => {
        const prompt = vi.fn().mockResolvedValue(undefined);
        const event = new Event("beforeinstallprompt", { cancelable: true });
        Object.assign(event, { prompt });

        const { rerender } = render(<PWAInstallProvider>{null}</PWAInstallProvider>);

        fireEvent(window, event);
        expect(event.defaultPrevented).toBe(true);

        rerender(
            <PWAInstallProvider>
                <InstallAppAction />
            </PWAInstallProvider>,
        );

        const installButton = await screen.findByRole("button", {
            name: "Install app",
        });
        fireEvent.click(installButton);
        await waitFor(() => expect(prompt).toHaveBeenCalledOnce());

        fireEvent(window, new Event("appinstalled"));
        await waitFor(() => {
            expect(
                screen.queryByRole("button", { name: "Install app" }),
            ).not.toBeInTheDocument();
        });
    });

    it("gives iOS Safari users Home Screen instructions", async () => {
        Object.defineProperty(navigator, "userAgent", {
            configurable: true,
            value: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)",
        });

        render(
            <PWAInstallProvider>
                <InstallAppAction />
            </PWAInstallProvider>,
        );

        fireEvent.click(
            await screen.findByRole("button", { name: "Install app" }),
        );
        expect(
            screen.getByRole("dialog", { name: "Add Expense Tracker to your Home Screen" }),
        ).toHaveTextContent("Choose Add to Home Screen.");
    });
});
