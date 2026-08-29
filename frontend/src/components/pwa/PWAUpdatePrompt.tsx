import { useRegisterSW } from "virtual:pwa-register/react";

const PWAUpdatePrompt = () => {
    const {
        needRefresh: [needRefresh, setNeedRefresh],
        offlineReady: [offlineReady, setOfflineReady],
        updateServiceWorker,
    } = useRegisterSW();

    if (!needRefresh && !offlineReady) {
        return null;
    }

    const updateMessage = needRefresh
        ? "A newer version is ready. Reload when you are ready to update."
        : "The app shell is ready for offline use. Expense data still needs a connection.";

    return (
        <aside className="pwa-update-prompt" role="status" aria-live="polite">
            <p>{updateMessage}</p>
            <div className="pwa-update-prompt__actions">
                {needRefresh ? (
                    <button
                        type="button"
                        className="ui-button ui-button-primary ui-button-sm"
                        onClick={() => void updateServiceWorker(true)}
                    >
                        Reload now
                    </button>
                ) : null}
                <button
                    type="button"
                    className="ui-button ui-button-ghost ui-button-sm"
                    onClick={() => {
                        setNeedRefresh(false);
                        setOfflineReady(false);
                    }}
                >
                    Later
                </button>
            </div>
        </aside>
    );
};

export default PWAUpdatePrompt;
