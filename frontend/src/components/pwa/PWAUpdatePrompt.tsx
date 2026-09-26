import { useCallback, useEffect, useState } from "react";
import { useRegisterSW } from "virtual:pwa-register/react";
import { usePWAUpdateSafety } from "../../hooks/usePWAUpdateSafety";

const UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000;

const PWAUpdatePrompt = () => {
    const [registration, setRegistration] =
        useState<ServiceWorkerRegistration>();
    const [isUpdateDismissed, setIsUpdateDismissed] = useState(false);
    const [isReloading, setIsReloading] = useState(false);
    const [activationFailed, setActivationFailed] = useState(false);
    const {
        canAutoApply,
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

            setIsUpdateDismissed(false);
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

    const applyUpdate = useCallback(async (automatic: boolean) => {
        if (isReloading) return;
        if (automatic && !claimActivation()) return;
        setIsReloading(true);
        setActivationFailed(false);
        try {
            await updateServiceWorker(true);
            setNeedRefresh(false);
            setIsUpdateDismissed(true);
        } catch {
            setIsReloading(false);
            setActivationFailed(true);
            releaseActivation();
        }
    }, [
        claimActivation,
        isReloading,
        releaseActivation,
        setNeedRefresh,
        updateServiceWorker,
    ]);

    useEffect(() => {
        if (
            !needRefresh ||
            !canAutoApply ||
            !isUpdateSafe ||
            isReloading ||
            activationFailed
        ) {
            return;
        }
        void applyUpdate(true);
    }, [
        activationFailed,
        applyUpdate,
        canAutoApply,
        isReloading,
        isUpdateSafe,
        needRefresh,
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
            ? "The update could not be applied automatically. You can retry it now."
            : isReloading
                ? "Applying the latest version…"
                : "A newer version is ready. Finish your current changes, or reload now."
        : "The app shell is ready for offline use. Expense data still needs a connection.";

    return (
        <aside className="pwa-update-prompt" role="status" aria-live="polite">
            <p>{updateMessage}</p>
            <div className="pwa-update-prompt__actions">
                {showUpdatePrompt ? (
                    <button
                        type="button"
                        className="ui-button ui-button-primary ui-button-sm"
                        onClick={() => void applyUpdate(false)}
                        disabled={isReloading}
                    >
                        {isReloading ? "Applying…" : activationFailed ? "Retry update" : "Reload now"}
                    </button>
                ) : null}
                <button
                    type="button"
                    className="ui-button ui-button-ghost ui-button-sm"
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
