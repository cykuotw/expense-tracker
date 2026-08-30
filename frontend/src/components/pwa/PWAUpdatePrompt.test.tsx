import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import PWAUpdatePrompt from "./PWAUpdatePrompt";

type RegisterCallback = (
    serviceWorkerUrl: string,
    registration: ServiceWorkerRegistration | undefined
) => void;

type RegisterOptions = {
    onRegisteredSW?: RegisterCallback;
};

const { updateServiceWorkerMock, useRegisterSWMock } = vi.hoisted(() => ({
    updateServiceWorkerMock: vi.fn(),
    useRegisterSWMock: vi.fn(),
}));

vi.mock("virtual:pwa-register/react", () => ({
    useRegisterSW: (options?: RegisterOptions) => useRegisterSWMock(options),
}));

const oneHour = 60 * 60 * 1000;
let registeredCallback: RegisterCallback | undefined;
let visibilityState: DocumentVisibilityState;

describe("PWAUpdatePrompt", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        visibilityState = "visible";
        Object.defineProperty(document, "visibilityState", {
            configurable: true,
            get: () => visibilityState,
        });
        registeredCallback = undefined;
        updateServiceWorkerMock.mockReset();
        useRegisterSWMock.mockImplementation((options?: RegisterOptions) => {
            registeredCallback = options?.onRegisteredSW;
            return {
                needRefresh: [true, vi.fn()],
                offlineReady: [false, vi.fn()],
                updateServiceWorker: updateServiceWorkerMock,
            };
        });
    });

    afterEach(() => {
        cleanup();
        vi.useRealTimers();
        delete (document as unknown as Record<string, unknown>).visibilityState;
    });

    it("checks for updates at registration, on foreground, and hourly", async () => {
        const update = vi.fn().mockResolvedValue(undefined);
        const registration = {
            update,
        } as unknown as ServiceWorkerRegistration;

        render(<PWAUpdatePrompt />);

        await act(async () => {
            registeredCallback?.("/sw.js", registration);
        });
        expect(update).toHaveBeenCalledTimes(1);

        await act(async () => {
            vi.advanceTimersByTime(oneHour);
        });
        expect(update).toHaveBeenCalledTimes(2);

        visibilityState = "hidden";
        fireEvent(document, new Event("visibilitychange"));
        expect(update).toHaveBeenCalledTimes(2);

        visibilityState = "visible";
        fireEvent(document, new Event("visibilitychange"));
        expect(update).toHaveBeenCalledTimes(3);
    });

    it("shows a dismissed update again when the app returns to the foreground", async () => {
        const registration = {
            update: vi.fn().mockResolvedValue(undefined),
        } as unknown as ServiceWorkerRegistration;

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration);
        });

        fireEvent.click(screen.getByRole("button", { name: "Later" }));
        expect(
            screen.queryByText("A newer version is ready. Reload when you are ready to update.")
        ).not.toBeInTheDocument();

        visibilityState = "visible";
        fireEvent(document, new Event("visibilitychange"));
        expect(
            screen.getByText("A newer version is ready. Reload when you are ready to update.")
        ).toBeVisible();
    });

    it("cleans up update checks when the prompt unmounts", async () => {
        const update = vi.fn().mockResolvedValue(undefined);
        const registration = {
            update,
        } as unknown as ServiceWorkerRegistration;
        const { unmount } = render(<PWAUpdatePrompt />);

        await act(async () => {
            registeredCallback?.("/sw.js", registration);
        });
        unmount();

        await act(async () => {
            vi.advanceTimersByTime(oneHour);
        });
        fireEvent(document, new Event("visibilitychange"));
        expect(update).toHaveBeenCalledTimes(1);
    });

    it("lets the user explicitly apply a waiting update", () => {
        render(<PWAUpdatePrompt />);

        fireEvent.click(screen.getByRole("button", { name: "Reload now" }));
        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
    });
});
