import { useState } from "react";

import ReceiptEditor from "@/components/receipt/ReceiptEditor";
import type { PreparedReceipt } from "@/lib/receiptEditor";

export default function ReceiptEditorLab() {
    const [result, setResult] = useState<PreparedReceipt | null>(null);

    return (
        <main className="page-shell">
            <div className="page-container max-w-6xl">
                <header className="mb-6 grid gap-2">
                    <div className="page-eyebrow">Development fixture lab</div>
                    <h1 className="page-title">Receipt editor</h1>
                    <p className="page-copy max-w-3xl">
                        This local-only page exercises receipt preparation without calling OCR,
                        saving an expense, uploading an image, or retaining browser data.
                    </p>
                </header>
                <ReceiptEditor
                    onPrepared={setResult}
                    onManualEntry={() => setResult(null)}
                />
                {result ? (
                    <p className="mt-4 rounded-2xl border border-success/30 bg-success/10 px-4 py-3 text-sm font-medium text-foreground" role="status">
                        Prepared JPEG is ready for the Phase 6 integration contract ({result.width} × {result.height}). No network request was made.
                    </p>
                ) : null}
            </div>
        </main>
    );
}
