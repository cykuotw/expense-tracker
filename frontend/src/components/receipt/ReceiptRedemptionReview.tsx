import { Button } from "@/components/ui/button";
import { useId } from "react";

interface Props {
    note?: string;
    conflict?: boolean;
    printedRedemption?: string;
    printedTotal: string;
    redemption?: string;
    net?: string | null;
    currency: string;
    onChange: (value: string | undefined) => void;
}

export default function ReceiptRedemptionReview({
    note, conflict, printedRedemption, printedTotal, redemption, net, currency, onChange,
}: Props) {
    const hintId = useId();
    return (
        <div className="mt-4 grid gap-2">
            {note && <p>Scan note: {note}</p>}
            {printedRedemption !== undefined && <p>Printed loyalty redemption: {printedRedemption || "Unreadable"}</p>}
            {note && conflict && <p role="alert">The scan note conflicts with the printed redemption. Confirm the amount to use.</p>}
            {redemption === undefined ? (
                <Button type="button" variant="outline" onClick={() => onChange("")}>Add loyalty redemption</Button>
            ) : (
                <fieldset className="grid gap-2">
                    <legend>Loyalty redemption</legend>
                    <label>Confirmed redemption amount
                        <input className="block rounded-md border p-2" inputMode="decimal" value={redemption}
                            aria-describedby={hintId} aria-invalid={redemption !== "" && net == null}
                            onChange={event => onChange(event.target.value)} />
                    </label>
                    <div className="flex flex-wrap gap-2">
                        <Button type="button" variant="outline" onClick={() => onChange(printedTotal)}>Redeem full total</Button>
                        <Button type="button" variant="outline" onClick={() => onChange(undefined)}>Remove redemption</Button>
                    </div>
                    <p id={hintId} aria-live="polite">{net == null ? "Enter an amount between zero and the printed total." : `Out of pocket: ${net} ${currency}`}</p>
                </fieldset>
            )}
        </div>
    );
}
