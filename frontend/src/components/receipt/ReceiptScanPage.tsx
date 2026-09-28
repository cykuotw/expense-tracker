import { lazy, Suspense } from "react";
import { Navigate, useNavigate } from "react-router-dom";

import { Spinner } from "@/components/ui/spinner";
import { useCreateExpense } from "@/hooks/CreateExpenseContextHooks";
import type { PreparedReceipt } from "@/lib/receiptEditor";

const ReceiptEditor = lazy(() => import("./ReceiptEditor"));

export default function ReceiptScanPage() {
    const navigate = useNavigate();
    const {
        selectedGroupId,
        receiptOCREnabled,
        startReceiptOCR,
        clearReceiptWorkflow,
    } = useCreateExpense();
    const query = selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : "";
    const returnTo = `/create_expense${query}`;

    if (receiptOCREnabled === false) return <Navigate to={returnTo} replace />;

    const leave = () => {
        clearReceiptWorkflow();
        navigate(returnTo, { replace: true });
    };
    const prepared = (receipt: PreparedReceipt) => {
        void startReceiptOCR(receipt);
        navigate(`/create_expense/receipt/review${query}`);
    };

    return (
        <main className="page-shell">
            <div className="page-container max-w-6xl">
                <Suspense fallback={<div className="panel-card grid min-h-72 place-items-center rounded-[2rem]" role="status"><Spinner className="size-8" aria-hidden="true" /><span className="sr-only">Loading receipt editor</span></div>}>
                    <ReceiptEditor onPrepared={prepared} onCancel={leave} onManualEntry={leave} />
                </Suspense>
            </div>
        </main>
    );
}
