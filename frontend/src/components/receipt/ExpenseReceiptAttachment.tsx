import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { apiFetch, getResponseErrorMessage } from "@/lib/api";
import { requestReceiptDraftWithRetention } from "@/lib/receiptOcr";
import { useAuth } from "@/hooks/AuthContextHooks";
import type { PreparedReceipt } from "@/lib/receiptEditor";
import type { ExpenseDetailData } from "@/types/expense";
import type { AccountSettingsData } from "@/types/account";

const ReceiptEditor = lazy(() => import("./ReceiptEditor"));
type ReceiptState = ExpenseDetailData["receipt"];
type Mutation = { key: string; body: { keep: boolean; token?: string } };

export default function ExpenseReceiptAttachment({ expenseId, initialReceipt }: {
    expenseId: string;
    initialReceipt: ReceiptState;
}) {
    const { userID } = useAuth();
    const [receipt, setReceipt] = useState<ReceiptState>(initialReceipt);
    const [available, setAvailable] = useState(false);
    const [editing, setEditing] = useState(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const mutationRef = useRef<Mutation | null>(null);

    useEffect(() => { setReceipt(initialReceipt); }, [initialReceipt]);
    useEffect(() => {
        const controller = new AbortController();
        void apiFetch("/account", { signal: controller.signal })
            .then(async (response) => response.ok ? await response.json() as AccountSettingsData : null)
            .then((account) => { if (!controller.signal.aborted) setAvailable(account?.capabilities?.receiptRetention === true); })
            .catch(() => { if (!controller.signal.aborted) setAvailable(false); });
        return () => controller.abort();
    }, []);

    const submit = async (mutation: Mutation) => {
        setBusy(true);
        setError(null);
        mutationRef.current = mutation;
        try {
            const response = await apiFetch(`/expense/${expenseId}/receipt`, {
                method: "PUT",
                headers: { "Content-Type": "application/json", "Idempotency-Key": mutation.key },
                body: JSON.stringify(mutation.body),
            });
            if (!response.ok) {
                setError(await getResponseErrorMessage(response, "Receipt change could not be saved."));
                return;
            }
            const result = await response.json() as { receipt?: ReceiptState };
            if (response.status === 202) {
                setReceipt(result.receipt);
                setError("The receipt change is still finishing. Try again in a moment.");
                return;
            }
            mutationRef.current = null;
            setEditing(false);
            setReceipt(result.receipt?.status === "deleted" ? undefined : result.receipt);
            if (result.receipt?.status === "failed") {
                setError("The new photo could not be kept. Your expense is saved.");
            }
        } catch {
            setError("Connection lost while saving the receipt. Try again to check the result.");
        } finally {
            setBusy(false);
        }
    };
    const prepared = async (photo: PreparedReceipt) => {
        if (!userID) { setError("Sign in again to keep this receipt."); return; }
        setBusy(true);
        setError(null);
        try {
            const result = await requestReceiptDraftWithRetention(photo, userID, true);
            if (!result.receiptToken) throw new Error("The photo was not stored.");
            await submit({ key: crypto.randomUUID(), body: { keep: true, token: result.receiptToken } });
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "Could not prepare the receipt photo.");
        } finally {
            setBusy(false);
        }
    };

    if (!available && !receipt) return null;
    return (
        <section className="mt-6 rounded-2xl border border-border bg-muted/20 p-4" aria-label="Receipt photo">
            <h2 className="font-semibold">Receipt photo</h2>
            <p className="mt-1 text-sm text-muted-foreground" role="status" aria-live="polite">
                {busy ? "Saving receipt change…" : receipt ? `Status: ${receipt.status}. The photo is private and cannot be previewed yet.` : "No photo is kept with this expense."}
            </p>
            {error ? <p className="mt-2 text-sm text-destructive" role="alert">{error}</p> : null}
            <div className="mt-3 flex flex-wrap gap-2">
                {available && !editing ? (
                    <button type="button" className="ui-button ui-button-outline min-h-11" disabled={busy} onClick={() => setEditing(true)}>
                        {receipt ? "Replace photo" : "Add photo"}
                    </button>
                ) : null}
                {receipt && receipt.status !== "deleting" ? (
                    <button type="button" className="ui-button ui-button-outline min-h-11" disabled={busy} onClick={() => void submit({ key: crypto.randomUUID(), body: { keep: false } })}>
                        Remove photo
                    </button>
                ) : null}
                {mutationRef.current ? (
                    <button type="button" className="ui-button ui-button-outline min-h-11" disabled={busy} onClick={() => void submit(mutationRef.current!)}>
                        Try again
                    </button>
                ) : receipt && (receipt.status === "pending" || receipt.status === "deleting") ? (
                    <button type="button" className="ui-button ui-button-outline min-h-11" disabled={busy} onClick={() => {
                        setBusy(true);
                        void apiFetch(`/expense/${expenseId}/receipt/reconcile`, { method: "POST" })
                            .then(async (response) => {
                                if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not finish the receipt change."));
                                const result = await response.json() as { receipt?: ReceiptState };
                                setReceipt(result.receipt?.status === "deleted" ? undefined : result.receipt);
                                setError(response.status === 202 ? "The receipt change is still finishing. Try again in a moment." : null);
                            })
                            .catch((reason) => setError(reason instanceof Error ? reason.message : "Could not finish the receipt change."))
                            .finally(() => setBusy(false));
                    }}>
                        Finish receipt change
                    </button>
                ) : null}
            </div>
            {editing ? (
                <div className="mt-4">
                    <p className="mb-3 text-sm text-muted-foreground">Crop or cover private details first. Expense values stay as they are.</p>
                    <Suspense fallback={<p>Loading photo editor…</p>}>
                        <ReceiptEditor onPrepared={(photo) => void prepared(photo)} onCancel={() => setEditing(false)} onManualEntry={() => setEditing(false)} hideLibraryChoice />
                    </Suspense>
                </div>
            ) : null}
        </section>
    );
}
