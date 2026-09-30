import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import PWAUpdatePrompt from "./PWAUpdatePrompt";

type RegisterCallback = (
    serviceWorkerUrl: string,
    registration: ServiceWorkerRegistration | undefined
) => void;

type RegisterOptions = {
    immediate?: boolean;
    onRegisteredSW?: RegisterCallback;
};

const {
    claimActivationMock,
    releaseActivationMock,
    setNeedRefreshMock,
    updateServiceWorkerMock,
    usePWAUpdateSafetyMock,
    useRegisterSWMock,
} = vi.hoisted(() => ({
    claimActivationMock: vi.fn(),
    releaseActivationMock: vi.fn(),
    setNeedRefreshMock: vi.fn(),
    updateServiceWorkerMock: vi.fn(),
    usePWAUpdateSafetyMock: vi.fn(),
    useRegisterSWMock: vi.fn(),
}));

vi.mock("virtual:pwa-register/react", () => ({
    useRegisterSW: (options?: RegisterOptions) => useRegisterSWMock(options),
}));
vi.mock("../../hooks/usePWAUpdateSafety", () => ({
    usePWAUpdateSafety: () => usePWAUpdateSafetyMock(),
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
        claimActivationMock.mockReset();
        claimActivationMock.mockReturnValue(true);
        releaseActivationMock.mockReset();
        usePWAUpdateSafetyMock.mockReturnValue({
            canAutoApply: false,
            isUpdateSafe: true,
            claimActivation: claimActivationMock,
            releaseActivation: releaseActivationMock,
        });
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
        expect(useRegisterSWMock).toHaveBeenCalledWith(
            expect.objectContaining({ immediate: true }),
        );

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

    it("keeps Later dismissed through foreground and hourly checks", async () => {
        const update = vi.fn().mockResolvedValue(undefined);
        const registration = {
            update,
        } as unknown as ServiceWorkerRegistration;

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration);
        });

        fireEvent.click(screen.getByRole("button", { name: "Later" }));
        expect(screen.queryByRole("button", { name: "Update now" })).not.toBeInTheDocument();

        await act(async () => {
            vi.advanceTimersByTime(oneHour);
        });
        visibilityState = "visible";
        fireEvent(document, new Event("visibilitychange"));

        expect(update).toHaveBeenCalledTimes(3);
        expect(screen.queryByRole("button", { name: "Update now" })).not.toBeInTheDocument();
        expect(updateServiceWorkerMock).not.toHaveBeenCalled();
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

        fireEvent.click(screen.getByRole("button", { name: "Update now" }));

        expect(screen.getByRole("button", { name: "Updating…" })).toBeDisabled();
        await act(async () => resolveUpdate?.());
    });

    it("clears the update prompt after a successful reload request", async () => {
        updateServiceWorkerMock.mockResolvedValue(undefined);
        render(<PWAUpdatePrompt />);

        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        });

        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
        expect(setNeedRefreshMock).toHaveBeenCalledWith(false);
        expect(
            screen.queryByText("A newer version is ready. Update now or keep using the app.")
        ).not.toBeInTheDocument();
    });

    it("keeps the update prompt available when the reload request fails", async () => {
        updateServiceWorkerMock.mockRejectedValue(new Error("update failed"));
        render(<PWAUpdatePrompt />);

        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        });

        expect(setNeedRefreshMock).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Retry update" })).toBeEnabled();
    });

    it("does not interrupt an active page when every tab is safe", async () => {
        render(<PWAUpdatePrompt />);

        await act(async () => undefined);
        expect(updateServiceWorkerMock).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Update now" })).toBeEnabled();
    });

    it("keeps the update action unavailable while an editing flow blocks it", async () => {
        usePWAUpdateSafetyMock.mockReturnValue({
            canAutoApply: true,
            isUpdateSafe: false,
            claimActivation: claimActivationMock,
            releaseActivation: releaseActivationMock,
        });

        render(<PWAUpdatePrompt />);

        expect(updateServiceWorkerMock).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Update now" })).toBeDisabled();
        expect(screen.getByText("A newer version is ready. Finish changes in all open tabs before updating.")).toBeVisible();
    });

    it("can apply the downloaded update when the freshness check is offline", async () => {
        const registration = {
            waiting: {} as ServiceWorker,
            installing: null,
            update: vi.fn().mockRejectedValue(new Error("offline")),
        };
        updateServiceWorkerMock.mockResolvedValue(undefined);

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration as unknown as ServiceWorkerRegistration);
        });
        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        });

        expect(registration.update).toHaveBeenCalledTimes(2);
        expect(claimActivationMock).toHaveBeenCalledTimes(1);
        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
    });

    it("uses the installed worker if a newer installation stalls", async () => {
        const installing = Object.assign(new EventTarget(), {
            state: "installing",
        }) as unknown as ServiceWorker;
        const registration = {
            waiting: {} as ServiceWorker,
            installing,
            update: vi.fn().mockResolvedValue(undefined),
        };
        updateServiceWorkerMock.mockResolvedValue(undefined);

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration as unknown as ServiceWorkerRegistration);
        });
        fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        await act(async () => undefined);
        expect(updateServiceWorkerMock).not.toHaveBeenCalled();

        await act(async () => {
            vi.advanceTimersByTime(20_000);
        });
        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
    });

    it("keeps the prompt available when no waiting worker remains", async () => {
        const registration = {
            waiting: null,
            installing: null,
            update: vi.fn().mockResolvedValue(undefined),
        };

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration as unknown as ServiceWorkerRegistration);
        });
        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        });

        expect(updateServiceWorkerMock).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Retry update" })).toBeEnabled();
    });

    it("checks for a newer deployment before applying the waiting worker", async () => {
        const installing = Object.assign(new EventTarget(), {
            state: "installing",
        }) as unknown as ServiceWorker;
        const registration = {
            installing: null as ServiceWorker | null,
            waiting: {} as ServiceWorker,
            update: vi.fn(),
        };
        registration.update.mockResolvedValueOnce(registration);
        registration.update.mockImplementationOnce(async () => {
            registration.installing = installing;
            return registration;
        });
        updateServiceWorkerMock.mockResolvedValue(undefined);

        render(<PWAUpdatePrompt />);
        await act(async () => {
            registeredCallback?.("/sw.js", registration as unknown as ServiceWorkerRegistration);
        });
        expect(registration.update).toHaveBeenCalledTimes(1);

        fireEvent.click(screen.getByRole("button", { name: "Update now" }));
        expect(registration.update).toHaveBeenCalledTimes(2);
        expect(screen.getByText("Checking for the newest version…")).toBeVisible();
        expect(updateServiceWorkerMock).not.toHaveBeenCalled();

        await act(async () => {
            (installing as unknown as { state: string }).state = "installed";
            installing.dispatchEvent(new Event("statechange"));
        });
        expect(updateServiceWorkerMock).toHaveBeenCalledWith(true);
    });
});
