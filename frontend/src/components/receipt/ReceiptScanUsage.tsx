import type { OCRScanUsage } from "@/types/ocr";

const tokenFormat = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 });
const costFormat = new Intl.NumberFormat("en-US", { minimumFractionDigits: 6, maximumFractionDigits: 6 });

export default function ReceiptScanUsage({ usage }: { usage: OCRScanUsage }) {
    const cost = usage.estimatedCostUsd;
    const estimate = cost === null ? "Estimate unavailable"
        : cost > 0 && cost < 0.000001 ? "Less than US$0.000001"
            : `US$${costFormat.format(cost)}`;
    return (
        <details className="panel-card group rounded-2xl p-4 sm:p-6">
            <summary className="flex min-h-11 cursor-pointer items-center justify-between gap-3 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
                <span className="text-sm font-medium">Scan usage</span>
                <span className="flex items-center gap-3 text-right text-sm text-muted-foreground">{estimate}<span aria-hidden="true"><span className="group-open:hidden">+</span><span className="hidden group-open:inline">−</span></span></span>
            </summary>
            <div className="mt-3 grid gap-3 text-sm">
                <p className="text-muted-foreground">Estimated Gemini API cost for this scan in USD, separate from your expense amount. Based on standard paid pricing; actual billing may differ.</p>
                <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-2">
                    <dt className="text-muted-foreground">Model</dt>
                    <dd className="max-w-48 break-all text-right">{usage.model}</dd>
                    <dt className="text-muted-foreground">Estimated API cost</dt>
                    <dd className="text-right tabular-nums">{estimate}</dd>
                    {usage.tokens ? [
                        ["Input tokens (photo + prompt)", usage.tokens.input],
                        ["Cached input (included above)", usage.tokens.cachedInput],
                        ["Output tokens", usage.tokens.output],
                        ["Thinking tokens", usage.tokens.thinking],
                        ["Total tokens", usage.tokens.total],
                    ].map(([label, count]) => <div key={label} className="contents">
                        <dt className="text-muted-foreground">{label}</dt>
                        <dd className="text-right tabular-nums">{tokenFormat.format(Number(count))}</dd>
                    </div>) : null}
                </dl>
                {!usage.tokens ? <p className="text-muted-foreground">Token usage was not available for this scan.</p> : null}
                {usage.tokens && cost === null ? <p className="text-muted-foreground">A cost estimate is not available for this scan.</p> : null}
            </div>
        </details>
    );
}
