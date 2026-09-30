import { lazy, Suspense, useCallback, useEffect, useState, type ChangeEvent } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiArrowLeft, mdiImagePlusOutline } from "@mdi/js";

import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useCreateExpense } from "@/hooks/CreateExpenseContextHooks";
import type { PreparedReceipt } from "@/lib/receiptEditor";
import ReceiptDraftReview from "./ReceiptDraftReview";

const ReceiptEditor = lazy(() => import("./ReceiptEditor"));

export default function ReceiptScanPage() {
    const navigate = useNavigate();
    const {
        selectedGroupId,
        receiptOCREnabled,
        startReceiptOCR,
        clearReceiptWorkflow,
    } = useCreateExpense();
    const [reviewOpen, setReviewOpen] = useState(false);
    const [photoLoaded, setPhotoLoaded] = useState(false);
    const [replacement, setReplacement] = useState<{ id: string; file: File } | null>(null);
    const [photoURL, setPhotoURL] = useState<string | null>(null);
    const query = selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : "";
    const returnTo = `/create_expense${query}`;
    const markPhotoLoaded = useCallback(() => setPhotoLoaded(true), []);

    useEffect(() => () => {
        if (photoURL) URL.revokeObjectURL(photoURL);
    }, [photoURL]);

    if (receiptOCREnabled === false && !reviewOpen) return <Navigate to={returnTo} replace />;

    const leave = () => {
        clearReceiptWorkflow();
        navigate(returnTo, { replace: true });
    };
    const editPhoto = () => {
        clearReceiptWorkflow();
        setPhotoURL(null);
        setReviewOpen(false);
    };
    const replacePhoto = (event: ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        event.target.value = "";
        if (!file) return;
        if (photoLoaded || reviewOpen) clearReceiptWorkflow();
        setPhotoURL(null);
        setReviewOpen(false);
        setPhotoLoaded(false);
        setReplacement({ id: crypto.randomUUID(), file });
    };
    const prepared = (receipt: PreparedReceipt) => {
        setPhotoURL(URL.createObjectURL(receipt.blob));
        setReviewOpen(true);
        void startReceiptOCR(receipt);
    };

    return (
        <div className="receipt-workspace-page">
            <header className="receipt-workspace-header">
                <div className="page-container flex max-w-3xl items-center justify-between gap-2">
                    <Button type="button" variant="ghost" className="min-h-11 px-2" onClick={leave}>
                        <Icon path={mdiArrowLeft} size={0.85} data-icon="inline-start" aria-hidden="true" />Back
                    </Button>
                    <span className="min-w-0 truncate text-sm font-semibold">Receipt</span>
                    <Button asChild variant="outline" className="min-h-11 shrink-0 cursor-pointer px-3">
                        <label>
                            <Icon path={mdiImagePlusOutline} size={0.8} data-icon="inline-start" aria-hidden="true" />
                            {photoLoaded ? "Change photo" : "Upload photo"}
                            <input className="sr-only" type="file" accept="image/*" onChange={replacePhoto} />
                        </label>
                    </Button>
                </div>
            </header>
            <div hidden={reviewOpen}>
                <main className="page-shell">
                    <div className="page-container max-w-3xl">
                        <Suspense fallback={<div className="panel-card grid min-h-72 place-items-center rounded-[2rem]" role="status"><Spinner className="size-8" aria-hidden="true" /><span className="sr-only">Loading receipt editor</span></div>}>
                            <ReceiptEditor
                                key={replacement?.id ?? "initial"}
                                initialFile={replacement?.file}
                                onPhotoLoaded={markPhotoLoaded}
                                onPhotoCleared={() => setPhotoLoaded(false)}
                                hideLibraryChoice
                                onPrepared={prepared}
                                onCancel={leave}
                                onManualEntry={leave}
                            />
                        </Suspense>
                    </div>
                </main>
            </div>
            {reviewOpen ? <ReceiptDraftReview photoURL={photoURL} onEditPhoto={editPhoto} /> : null}
        </div>
    );
}
