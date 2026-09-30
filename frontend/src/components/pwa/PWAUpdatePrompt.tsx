import { useCallback, useEffect, useState } from "react";
import { useRegisterSW } from "virtual:pwa-register/react";
import { usePWAUpdateSafety } from "../../hooks/usePWAUpdateSafety";

const UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000;
const UPDATE_INSTALL_TIMEOUT_MS = 20 * 1000;

function waitForInstallingWorker(registration: ServiceWorkerRegistration) {
    const installing = registration.installing;
    if (!installing) return Promise.resolve();
    if (installing.state === "installed" || installing.state === "activated") {
        return Promise.resolve();
    }
    if (installing.state === "redundant") {
        return Promise.reject(new Error("The newest update could not be installed."));
    }

    return new Promise<void>((resolve, reject) => {
        const stopWaiting = () => {
            window.clearTimeout(timeoutId);
            installing.removeEventListener("statechange", handleStateChange);
        };
        const handleStateChange = () => {
            if (installing.state === "installed" || installing.state === "activated") {
                stopWaiting();
                resolve();
            } else if (installing.state === "redundant") {
                stopWaiting();
                reject(new Error("The newest update could not be installed."));
            }
        };
        const timeoutId = window.setTimeout(() => {
            stopWaiting();
            reject(new Error("The newest update is taking too long to install."));
        }, UPDATE_INSTALL_TIMEOUT_MS);
        installing.addEventListener("statechange", handleStateChange);
        handleStateChange();
    });
}

const PWAUpdatePrompt = () => {
    const [registration, setRegistration] =
        useState<ServiceWorkerRegistration>();
    const [isUpdateDismissed, setIsUpdateDismissed] = useState(false);
    const [isReloading, setIsReloading] = useState(false);
    const [isChecking, setIsChecking] = useState(false);
    const [activationFailed, setActivationFailed] = useState(false);
    const {
        isUpdateSafe,
        claimActivation,
        releaseActivation,
    } = usePWAUpdateSafety();
    const {
        needRefresh: [needRefresh, setNeedRefresh],
        offlineReady: [offlineReady, setOfflineReady],
        updateServiceWorker,
    } = useRegisterSW({
        immediate: true,
        onRegisteredSW: (_serviceWorkerUrl, nextRegistration) => {
            setRegistration(nextRegistration);
        },
    });

    useEffect(() => {
        if (!registration) return;

        const checkForUpdate = () => {
            void registration.update().catch(() => undefined);
        };
        const handleVisibilityChange = () => {
            if (document.visibilityState !== "visible") return;

            checkForUpdate();
        };

        checkForUpdate();
        document.addEventListener("visibilitychange", handleVisibilityChange);
        const intervalId = window.setInterval(
            checkForUpdate,
            UPDATE_CHECK_INTERVAL_MS
        );

        return () => {
            document.removeEventListener(
                "visibilitychange",
                handleVisibilityChange
            );
            window.clearInterval(intervalId);
        };
    }, [registration]);

    const applyUpdate = useCallback(async () => {
        if (isReloading) return;
        setIsReloading(true);
        setIsChecking(true);
        setActivationFailed(false);
        try {
            // A newer deployment may have replaced the worker that first raised
            // the prompt. Wait for that install before activating the waiting worker.
            if (registration) {
                try {
                    await registration.update();
                } catch {
                    // A previously downloaded worker remains usable offline.
                }
                try {
                    await waitForInstallingWorker(registration);
                } catch {
                    // Use the last successfully installed waiting worker.
                }
                if (!registration.waiting) throw new Error("No update is ready.");
            }
            if (!claimActivation()) throw new Error("Another tab is not ready.");
            setIsChecking(false);
            await updateServiceWorker(true);
            setNeedRefresh(false);
            setIsUpdateDismissed(true);
        } catch {
            setIsChecking(false);
            setIsReloading(false);
            setActivationFailed(true);
            releaseActivation();
        }
    }, [
        claimActivation,
        isReloading,
        registration,
        releaseActivation,
        setNeedRefresh,
        updateServiceWorker,
    ]);

    useEffect(() => {
        if (!needRefresh) setActivationFailed(false);
    }, [needRefresh]);

    const showUpdatePrompt = needRefresh && !isUpdateDismissed;

    if (!showUpdatePrompt && !offlineReady) {
        return null;
    }

    const updateMessage = showUpdatePrompt
        ? activationFailed
            ? "The update could not be applied. You can retry it now."
            : isChecking
                ? "Checking for the newest version…"
                : isReloading
                    ? "Applying the latest version…"
                    : isUpdateSafe
                        ? "A newer version is ready. Update now or keep using the app."
                        : "A newer version is ready. Finish changes in all open tabs before updating."
        : "The app shell is ready for offline use. Expense data still needs a connection.";

    return (
        <aside className="pwa-update-prompt" role="status" aria-live="polite">
            <p>{updateMessage}</p>
            <div className="pwa-update-prompt__actions">
                {showUpdatePrompt ? (
                    <button
                        type="button"
                        className="ui-button ui-button-primary ui-button-sm min-h-11"
                        onClick={() => void applyUpdate()}
                        disabled={isReloading || !isUpdateSafe}
                    >
                        {isReloading ? "Updating…" : activationFailed ? "Retry update" : "Update now"}
                    </button>
                ) : null}
                <button
                    type="button"
                    className="ui-button ui-button-ghost ui-button-sm min-h-11"
                    disabled={isReloading}
                    onClick={() => {
                        if (needRefresh) {
                            setIsUpdateDismissed(true);
                        } else {
                            setOfflineReady(false);
                        }
                    }}
                >
                    Later
                </button>
            </div>
        </aside>
    );
};

export default PWAUpdatePrompt;
