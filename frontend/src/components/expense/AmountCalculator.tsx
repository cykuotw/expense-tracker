import { useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import Icon from "@mdi/react";
import { mdiCalculatorVariantOutline } from "@mdi/js";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Field, FieldLabel } from "../ui/field";
import { calculateAmount } from "../../lib/amountCalculator";

interface AmountCalculatorProps {
    amount: string;
    amountDigits: number | null;
    currency: string;
    onApply: (amount: string) => void;
}

const keypad = ["7", "8", "9", "÷", "4", "5", "6", "×", "1", "2", "3", "−", "0", ".", "⌫", "+"];

export default function AmountCalculator({ amount, amountDigits, currency, onApply }: AmountCalculatorProps) {
    const [open, setOpen] = useState(false);
    const [expression, setExpression] = useState("");
    const dialogRef = useRef<HTMLDialogElement>(null);
    const triggerRef = useRef<HTMLButtonElement>(null);
    const closeRef = useRef<HTMLButtonElement>(null);
    const id = useId();
    const result = calculateAmount(expression, amountDigits);

    useEffect(() => {
        if (!open) return;
        const dialog = dialogRef.current;
        const trigger = triggerRef.current;
        if (!dialog) return;
        const resize = () => {
            const viewport = window.visualViewport;
            dialog.style.setProperty("--calculator-top", `${viewport?.offsetTop ?? 0}px`);
            dialog.style.setProperty("--calculator-height", `${viewport?.height ?? window.innerHeight}px`);
        };
        resize();
        dialog.showModal();
        // Opening the keypad should not summon the software keyboard.
        closeRef.current?.focus({ preventScroll: true });
        window.addEventListener("resize", resize);
        window.visualViewport?.addEventListener("resize", resize);
        window.visualViewport?.addEventListener("scroll", resize);
        return () => {
            window.removeEventListener("resize", resize);
            window.visualViewport?.removeEventListener("resize", resize);
            window.visualViewport?.removeEventListener("scroll", resize);
            dialog.close();
            trigger?.focus({ preventScroll: true });
        };
    }, [open]);

    return <>
        <Button ref={triggerRef} type="button" variant="outline" size="icon" className="size-11 shrink-0"
            aria-label="Calculate amount" aria-haspopup="dialog" disabled={amountDigits === null}
            onClick={() => { setExpression(amount); setOpen(true); }}>
            <Icon path={mdiCalculatorVariantOutline} aria-hidden="true" />
        </Button>
        {open && createPortal(
            <dialog ref={dialogRef} className="amount-calculator" aria-labelledby={`${id}-title`} aria-describedby={`${id}-description`}
                onCancel={(event) => { event.preventDefault(); setOpen(false); }}
                onKeyDown={(event) => {
                    if (event.key === "Enter" && event.target instanceof HTMLInputElement) event.preventDefault();
                    if (event.key === "Tab") {
                        const controls = Array.from(event.currentTarget.querySelectorAll<HTMLElement>(
                            'button, input, [href], [tabindex="0"]',
                        )).filter((control) => control.tabIndex >= 0 && !("disabled" in control && control.disabled));
                        const destination = event.shiftKey && event.target === controls[0] ? controls.at(-1) :
                            !event.shiftKey && event.target === controls.at(-1) ? controls[0] : null;
                        if (destination) { event.preventDefault(); destination.focus({ preventScroll: true }); }
                    }
                }}>
                <div className="flex items-center justify-between gap-3">
                    <h2 id={`${id}-title`} className="text-lg font-semibold">Calculate amount</h2>
                    <Button ref={closeRef} type="button" variant="ghost" className="min-h-11" onClick={() => setOpen(false)}>Cancel</Button>
                </div>
                <p id={`${id}-description`} className="mt-2 text-sm text-muted-foreground">Combine amounts in {currency}. Apply the result when you're ready.</p>
                <Field className="mt-4">
                    <FieldLabel htmlFor={`${id}-expression`}>Calculation</FieldLabel>
                    <Input id={`${id}-expression`} type="text" inputMode="text" autoComplete="off" maxLength={256}
                        value={expression} onChange={(event) => setExpression(event.target.value)}
                        placeholder="12.50 + 8.25" aria-describedby={`${id}-result`} />
                </Field>
                <div id={`${id}-result`} className="mt-3 min-h-16 rounded-xl bg-muted p-3" aria-live="polite" aria-atomic="true">
                    {result.error ? <p className="text-sm text-destructive">{result.error}</p> :
                        result.amount ? <>
                            <p className="break-all text-xl font-semibold tabular-nums">{result.amount} {currency}</p>
                            {result.rounded ? <p className="mt-1 text-xs text-muted-foreground">Rounded to {amountDigits} decimal places for {currency}.</p> : null}
                        </> : <p className="text-sm text-muted-foreground">Enter amounts to calculate a total.</p>}
                </div>
                <div className="mt-3 grid grid-cols-4 gap-2">
                    {keypad.map((key) => <Button key={key} type="button" variant="outline" className="min-h-11"
                        aria-label={key === "⌫" ? "Backspace" : key === "÷" ? "Divide" : key === "×" ? "Multiply" : key === "−" ? "Subtract" : key === "+" ? "Add" : key}
                        onClick={() => setExpression((current) => key === "⌫" ? current.slice(0, -1) : (current + key).slice(0, 256))}>
                        {key}
                    </Button>)}
                </div>
                <div className="mt-4 flex gap-3">
                    <Button type="button" variant="outline" className="min-h-11" onClick={() => setExpression("")}>Clear</Button>
                    <Button type="button" className="min-h-11 flex-1" disabled={!result.amount} onClick={() => {
                        if (result.amount) { onApply(result.amount); setOpen(false); }
                    }}>Apply amount</Button>
                </div>
            </dialog>, document.body,
        )}
    </>;
}
