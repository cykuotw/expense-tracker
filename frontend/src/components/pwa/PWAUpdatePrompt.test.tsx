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

const { setNeedRefreshMock, updateServiceWorkerMock, useRegisterSWMock } = vi.hoisted(() => ({
    setNeedRefreshMock: vi.fn(),
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
        setNeedRefreshMock.mockReset();
        updateServiceWorkerMock.mockReset();
        useRegisterSWMock.mockImplementation((options?: RegisterOptions) => {
            registeredCallback = options?.onRegisteredSW;
            return {
                needRefresh: [true, setNeedRefreshMock],
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

    it("shows that an update is being applied", async () => {
        let resolveUpdate: (() => void) | undefined;
        updateServiceWorkerMock.mockImplementation(
            () => new Promise<void>((resolve) => {
                resolveUpdate = resolve;
            })
        );
        render(<PWAUpdatePrompt />);

        fireEvent.click(screen.getByRole("button", { name: "Reload now" }));

        expect(screen.getByRole("button", { name: "Reloading…" })).toBeDisabled();
        await act(async () => resolveUpdate?.());
    });

    it("clears the update prompt after a successful reload request", async () => {
        updateServiceWorkerMock.mockResolvedValue(undefined);
        render(<PWAUpdatePrompt />);

        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Reload now" }));
        });

        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
        expect(setNeedRefreshMock).toHaveBeenCalledWith(false);
        expect(
            screen.queryByText("A newer version is ready. Reload when you are ready to update.")
        ).not.toBeInTheDocument();
    });

    it("keeps the update prompt available when the reload request fails", async () => {
        updateServiceWorkerMock.mockRejectedValue(new Error("update failed"));
        render(<PWAUpdatePrompt />);

        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Reload now" }));
        });

        expect(setNeedRefreshMock).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Reload now" })).toBeEnabled();
    });
});
