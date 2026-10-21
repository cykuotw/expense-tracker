import { useEffect, useRef, useState } from "react";
import { apiFetch, getResponseErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";

export default function ReceiptViewer({ expenseId }: { expenseId: string }) {
    return <ReceiptImageView key={expenseId} expenseId={expenseId} />;
}

function ReceiptImageView({ expenseId }: { expenseId: string }) {
    const [url, setUrl] = useState("");
    const [busy, setBusy] = useState(false);
    const [message, setMessage] = useState("");
    const expiry = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
    const request = useRef<AbortController | null>(null);
    useEffect(() => () => { clearTimeout(expiry.current); request.current?.abort(); }, [expenseId]);
    const hide = () => { clearTimeout(expiry.current); request.current?.abort(); setUrl(""); setBusy(false); setMessage(""); };
    const reveal = async () => {
        clearTimeout(expiry.current);
        request.current?.abort();
        const controller = new AbortController(); request.current = controller;
        setBusy(true); setMessage(""); setUrl("");
        try {
            const response = await apiFetch(`/expense/${expenseId}/receipt/view`, { method: "POST", signal: controller.signal });
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not open this receipt. Try again."));
            const result = await response.json() as { url: string; expiresAt: string };
            const lifetime = Date.parse(result.expiresAt) - Date.now();
            if (typeof result.url !== "string" || !result.url.startsWith("https://") || !Number.isFinite(lifetime) || lifetime <= 0) throw new Error("The receipt link expired. Try again.");
            if (controller.signal.aborted) return;
            setUrl(result.url);
            clearTimeout(expiry.current);
            expiry.current = setTimeout(() => { setUrl(""); setMessage("The receipt view expired. Open it again to continue."); }, Math.min(lifetime, 60_000));
        } catch (error) {
            if (!controller.signal.aborted) setMessage(error instanceof Error ? error.message : "Could not open this receipt.");
        } finally { if (!controller.signal.aborted) setBusy(false); }
    };
    return <div className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-2">
            <Button className="min-h-11" type="button" variant="outline" disabled={busy} onClick={() => void reveal()}>{busy ? "Opening receipt…" : url ? "Refresh receipt view" : "View receipt"}</Button>
            {url || busy ? <Button className="min-h-11" type="button" variant="ghost" onClick={hide}>Hide receipt</Button> : null}
        </div>
        <p className="text-sm text-muted-foreground">Photo metadata was removed. Only details you covered in the editor are masked.</p>
        {message ? <p role="status" className="text-sm text-muted-foreground">{message}</p> : null}
        {url ? <img src={url} alt="Saved receipt" referrerPolicy="no-referrer" className="max-h-[70vh] max-w-full rounded-lg object-contain" onError={() => { setUrl(""); setMessage("Could not load the receipt. Open it again to retry."); }} /> : null}
    </div>;
}
