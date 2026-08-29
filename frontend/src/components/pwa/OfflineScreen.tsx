const OfflineScreen = () => (
    <main className="pwa-status-screen" role="status" aria-live="polite">
        <div className="pwa-status-screen__card">
            <p className="section-label">Offline</p>
            <h1>You are not connected</h1>
            <p>
                Expense data needs a connection. We will check your session again
                when you are back online.
            </p>
        </div>
    </main>
);

export default OfflineScreen;
