import { useEffect, useState } from "react";
import { apiFetch, getResponseErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Field, FieldLabel, FieldDescription } from "@/components/ui/field";
import { Switch } from "@/components/ui/switch";

interface Preference { enabled: boolean; policyVersion: string; revision: number; purgePending: boolean; captureAvailable: boolean }
const POLICY = "ocr-improvement-v1";

export default function OCRImprovementSettings({ granted }: { granted: boolean }) {
    const [preference, setPreference] = useState<Preference | null>(null);
    const [enabled, setEnabled] = useState(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const [status, setStatus] = useState("");
    const load = async (signal?: AbortSignal) => {
        const response = await apiFetch("/account/ocr-improvement", { signal });
        if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not load receipt sharing preferences."));
        const data = await response.json() as Preference;
        if (!signal?.aborted) { setPreference(data); setEnabled(data.enabled); }
    };
    useEffect(() => {
        const controller = new AbortController();
        void load(controller.signal).catch(() => { if (!controller.signal.aborted) setError("Could not load receipt sharing preferences."); });
        return () => controller.abort();
    }, []);
    const save = async (purge = false) => {
        if (!preference) { setError(""); try { await load(); } catch { setError("Could not load receipt sharing preferences."); } return; }
        setBusy(true); setError(""); setStatus("");
        try {
            const response = await apiFetch(`/account/ocr-improvement${purge ? "/purge" : ""}`, {
                method: purge ? "POST" : "PATCH", headers: { "Content-Type": "application/json" },
                ...(purge ? {} : { body: JSON.stringify({ enabled, policyVersion: POLICY, expectedRevision: preference.revision }) }),
            });
            if (response.status === 409) {
                await load();
                throw new Error("Your sharing preference changed. Review the current setting, then try again.");
            }
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not save your preference. Try again."));
            const data = await response.json() as Preference;
            setPreference(data); setEnabled(data.enabled);
            setStatus(data.purgePending ? "Sharing is off. Evidence deletion is pending. Wait up to two minutes, then check deletion again." : "Preference saved.");
        } catch (reason) { setError(reason instanceof Error ? reason.message : "Could not save your preference."); }
        finally { setBusy(false); }
    };
    if (!granted && !preference?.enabled && !preference?.purgePending && (!preference || preference.revision === 0) && !error) return null;
    return <Card>
        <CardHeader><CardTitle>Help improve receipt scanning</CardTitle><CardDescription>Optional sharing, separate from keeping a receipt with an expense.</CardDescription></CardHeader>
        <CardContent className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">If you opt in, we can privately retain your processed receipt image, scanning response, draft, and confirmed receipt values to investigate extraction errors. Evidence is accessible only to authorized improvement tooling and has no automatic expiry after confirmation. Turning this off requests deletion of previous samples; your expenses and separately saved receipt photos stay intact.</p>
            <p className="text-sm text-muted-foreground">Scanning sends the processed image to Google Gemini whether sharing is on or off. This setting controls our improvement storage, not Google's data handling. <a href="https://ai.google.dev/gemini-api/terms" target="_blank" rel="noopener noreferrer" className="underline">Google's API terms</a> apply to the configured service tier.</p>
            {preference && !preference.captureAvailable ? <p className="text-sm text-muted-foreground">Improvement collection is currently disabled on this deployment. Your preference can be saved for when it becomes available.</p> : null}
            {preference ? <Field orientation="horizontal"><div><FieldLabel className="min-h-11 items-center" htmlFor="ocr-improvement-sharing">Share receipt samples for improvement</FieldLabel><FieldDescription>Off by default. You can turn this off at any time.</FieldDescription></div><Switch id="ocr-improvement-sharing" checked={enabled} onCheckedChange={setEnabled} disabled={busy || preference.purgePending || (!granted && !enabled)} /></Field> : null}
            {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
            {status ? <p role="status" className="text-sm text-muted-foreground">{status}</p> : null}
            {preference?.purgePending ? <><p role="status" className="text-sm text-muted-foreground">Evidence deletion is pending. Re-enabling sharing is unavailable until deletion completes.</p><Button className="min-h-11" type="button" variant="outline" disabled={busy} onClick={() => void save(true)}>{busy ? "Checking deletion…" : "Check deletion"}</Button></> : <Button className="min-h-11" type="button" variant="outline" disabled={busy || (!!preference && enabled === preference.enabled)} onClick={() => void save()}>{busy ? "Saving…" : preference ? "Save sharing preference" : "Retry loading preference"}</Button>}
        </CardContent>
    </Card>;
}
