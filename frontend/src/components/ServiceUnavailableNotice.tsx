import { useEffect, useState } from "react";
import { useAuth } from "../hooks/AuthContextHooks";
import { type OperatingHours, outsideOperatingHours, parseOperatingHours } from "../lib/operatingHours";
import { Button } from "./ui/button";
import { dismissServiceRecovery, useServiceUnavailable } from "../lib/serviceAvailability";

export default function ServiceUnavailableNotice({ initial = false }: { initial?: boolean }) {
    const { refreshSession } = useAuth();
    const unavailable = useServiceUnavailable();
    const [hours, setHours] = useState<OperatingHours | null>(null);
    const [checking, setChecking] = useState(false);
    const [now, setNow] = useState(() => new Date());

    useEffect(() => {
        let request: AbortController;
        const load = () => {
            request?.abort();
            request = new AbortController();
            const timeout = window.setTimeout(() => request.abort(), 5000);
            void fetch("/operating-hours.json", { cache: "no-store", signal: request.signal })
                .then(async (response) => {
                    if (response.ok) setHours(parseOperatingHours(await response.json()));
                }).catch(() => { /* Generic notice remains useful without policy. */ })
                .finally(() => window.clearTimeout(timeout));
        };
        load();
        const timer = window.setInterval(() => { setNow(new Date()); load(); }, 60_000);
        return () => { request.abort(); window.clearInterval(timer); };
    }, []);

    async function retry() {
        setChecking(true);
        try { await refreshSession(); } finally { setChecking(false); }
    }

    if (!unavailable) {
        return <section className="mx-auto flex max-w-3xl flex-col gap-3 p-4" role="status">
            <h2 className="text-lg font-semibold">Service is available again</h2>
            <p>Reload to refresh missing or outdated data. Copy any unsaved entries first. Your page has been kept open and no failed submissions have been replayed.</p>
            <div className="flex flex-wrap gap-3">
                <Button variant="outline" onClick={() => window.location.reload()}>Reload page</Button>
                <Button variant="ghost" onClick={dismissServiceRecovery}>Keep this page</Button>
            </div>
        </section>;
    }

    const offHours = hours?.enabled && outsideOperatingHours(hours, now);
    const content = (
        <section className={initial ? "pwa-status-screen__card" : "mx-auto flex max-w-3xl flex-col gap-3 p-4"} role="status" aria-live="polite">
            <h1 className="text-lg font-semibold">Service temporarily unavailable</h1>
            <p>{offHours
                ? "The service may be offline for its scheduled overnight shutdown."
                : "We could not reach the service. Please contact the system administrator if you need access."}</p>
            {hours?.enabled ? <p>Daily operating hours: {hours.start}–{hours.stop} ({hours.timezone}). Startup may take a few minutes. Contact the system administrator if access is needed.</p> : null}
            <p>Your session has not been cleared. Failed submissions are not retried automatically. Check whether a change was saved before submitting again. If a receipt attachment has expired, upload the photo again.</p>
            <Button variant="outline" disabled={checking} onClick={() => { void retry(); }}>
                {checking ? "Checking service…" : "Try again"}
            </Button>
        </section>
    );
    return initial ? <main className="pwa-status-screen">{content}</main> : content;
}
