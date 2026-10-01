import { lazy, Suspense, useCallback, useEffect, useRef, useState, type ChangeEvent } from "react";
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
        receiptRetentionAvailable,
        keepReceipt,
        setKeepReceipt,
        startReceiptOCR,
        clearReceiptWorkflow,
    } = useCreateExpense();
    const [reviewOpen, setReviewOpen] = useState(false);
    const [photoLoaded, setPhotoLoaded] = useState(false);
    const [replacement, setReplacement] = useState<{ id: string; file: File } | null>(null);
    const [photoURL, setPhotoURL] = useState<string | null>(null);
    const replacePhotoInputRef = useRef<HTMLInputElement>(null);
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
            <input ref={replacePhotoInputRef} className="sr-only" type="file" accept="image/*" aria-label="Receipt photo file" tabIndex={-1} onChange={replacePhoto} />
            <header className="receipt-workspace-header md:hidden">
                <div className="page-container flex max-w-3xl items-center justify-between gap-2">
                    <Button type="button" variant="ghost" className="min-h-11 px-2" onClick={leave}>
                        <Icon path={mdiArrowLeft} size={0.85} data-icon="inline-start" aria-hidden="true" />Back
                    </Button>
                    <span className="min-w-0 truncate text-sm font-semibold">Receipt</span>
                    <Button type="button" variant="outline" className="min-h-11 shrink-0 px-3" onClick={() => replacePhotoInputRef.current?.click()}>
                        <Icon path={mdiImagePlusOutline} size={0.8} data-icon="inline-start" aria-hidden="true" />
                        {photoLoaded ? "Change photo" : "Upload photo"}
                    </Button>
                </div>
            </header>
            <div className="hidden px-8 pt-8 md:block">
                <div className={reviewOpen ? "page-container max-w-3xl" : "page-container max-w-3xl lg:max-w-6xl"}>
                    <div className="desktop-page-utility">
                        <button type="button" className="desktop-back-link" onClick={leave}>
                            <span className="desktop-back-link__icon" aria-hidden="true"><Icon path={mdiArrowLeft} size={0.82} /></span>
                            Back to expense
                        </button>
                    </div>
                </div>
            </div>
            <div hidden={reviewOpen}>
                <main className="page-shell md:pt-0">
                    <div className="page-container max-w-3xl lg:max-w-6xl">
                        <header className="page-header desktop-page-header">
                            <div className="page-header__copy">
                                <div className="page-eyebrow">Receipt</div>
                                <h1 className="page-title">Receipt photo</h1>
                                <p className="page-copy">Crop the photo and cover private details before scanning.</p>
                            </div>
                            <Button type="button" variant="outline" className="min-h-11 shrink-0" onClick={() => replacePhotoInputRef.current?.click()}>
                                <Icon path={mdiImagePlusOutline} size={0.8} aria-hidden="true" />
                                {photoLoaded ? "Change photo" : "Upload photo"}
                            </Button>
                        </header>
                        {receiptRetentionAvailable ? (
                            <label className="panel-card mb-4 flex cursor-pointer items-start gap-3 rounded-2xl p-4">
                                <input
                                    type="checkbox"
                                    className="mt-1 size-5 shrink-0 accent-primary"
                                    checked={keepReceipt}
                                    onChange={(event) => setKeepReceipt(event.target.checked)}
                                />
                                <span className="min-w-0">
                                    <span className="block font-semibold">Keep receipt photo with expense</span>
                                    <span className="mt-1 block text-sm text-muted-foreground">Only the cropped and covered photo will be kept. Group members may be able to view it later. Leave this off to use scanned values only.</span>
                                </span>
                            </label>
                        ) : (
                            <p className="mb-4 text-sm text-muted-foreground">The photo is used to read values and is not kept with the expense.</p>
                        )}
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
