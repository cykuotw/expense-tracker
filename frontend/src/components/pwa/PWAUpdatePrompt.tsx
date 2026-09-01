import { useEffect, useState } from "react";
import { useRegisterSW } from "virtual:pwa-register/react";

const UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000;

const PWAUpdatePrompt = () => {
    const [registration, setRegistration] =
        useState<ServiceWorkerRegistration>();
    const [isUpdateDismissed, setIsUpdateDismissed] = useState(false);
    const [isReloading, setIsReloading] = useState(false);
    const {
        needRefresh: [needRefresh, setNeedRefresh],
        offlineReady: [offlineReady, setOfflineReady],
        updateServiceWorker,
    } = useRegisterSW({
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

    const showUpdatePrompt = needRefresh && !isUpdateDismissed;

    const applyUpdate = async () => {
        if (isReloading) return;
        setIsReloading(true);
        try {
            await updateServiceWorker(true);
            setNeedRefresh(false);
            setIsUpdateDismissed(true);
        } catch {
            setIsReloading(false);
        }
    };

    if (!showUpdatePrompt && !offlineReady) {
        return null;
    }

    const updateMessage = showUpdatePrompt
        ? "A newer version is ready. Reload when you are ready to update."
        : "The app shell is ready for offline use. Expense data still needs a connection.";

    return (
        <aside className="pwa-update-prompt" role="status" aria-live="polite">
            <p>{updateMessage}</p>
            <div className="pwa-update-prompt__actions">
                {showUpdatePrompt ? (
                    <button
                        type="button"
                        className="ui-button ui-button-primary ui-button-sm"
                        onClick={() => void applyUpdate()}
                        disabled={isReloading}
                    >
                        {isReloading ? "Reloading…" : "Reload now"}
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
