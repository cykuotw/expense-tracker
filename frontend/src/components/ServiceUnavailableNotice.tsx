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
            <h2 className="text-lg font-semibold">Service is back</h2>
            <p>Copy unsaved entries before reloading to refresh your data.</p>
            <div className="flex flex-wrap gap-3">
                <Button variant="outline" onClick={() => window.location.reload()}>Reload page</Button>
                <Button variant="ghost" onClick={dismissServiceRecovery}>Keep this page</Button>
            </div>
        </section>;
    }

    const offHours = hours?.enabled && outsideOperatingHours(hours, now);
    const content = (
        <section className={initial ? "pwa-status-screen__card flex flex-col gap-5" : "mx-auto flex max-w-3xl flex-col gap-5 p-4"} role="status" aria-live="polite">
            <h1 className="text-lg font-semibold">{offHours ? "Outside service hours" : "Service unavailable"}</h1>
            {hours?.enabled ? (
                <div>
                    <p className="text-sm text-muted-foreground">Daily operating hours</p>
                    <p className="text-2xl font-semibold tabular-nums">{hours.start}–{hours.stop}</p>
                    <p className="text-sm text-muted-foreground">{hours.timezone}</p>
                </div>
            ) : null}
            <p className="text-sm">{offHours
                ? "Need access now? Contact the administrator."
                : "Please try again or contact the administrator."}</p>
            <Button className="min-h-11 self-start" disabled={checking} onClick={() => { void retry(); }}>
                {checking ? "Checking…" : "Try again"}
            </Button>
            <details className="text-sm text-muted-foreground">
                <summary className="min-h-11 cursor-pointer content-center rounded-sm focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring">More details</summary>
                <div className="flex flex-col gap-2 pt-2">
                    {offHours ? <p>The service may be offline for its scheduled shutdown.</p> : null}
                    {hours?.enabled ? <p>Startup may take a few minutes after opening.</p> : null}
                    <p>Your session is kept. Failed submissions are not retried automatically. Check whether a change was saved before submitting again.</p>
                    <p>If a receipt attachment has expired, upload the photo again.</p>
                </div>
            </details>
        </section>
    );
    return initial ? <main className="pwa-status-screen">{content}</main> : content;
}
