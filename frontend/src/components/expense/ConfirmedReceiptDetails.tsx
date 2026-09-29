import Icon from "@mdi/react";
import { mdiArrowDown, mdiArrowUp, mdiDeleteOutline, mdiPlus } from "@mdi/js";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import type { ConfirmedExpenseItem } from "@/types/ocr";

interface ConfirmedReceiptDetailsProps {
    merchant: string;
    onMerchantChange: (value: string) => void;
    subtotal: string;
    onSubtotalChange: (value: string) => void;
    tax: string;
    onTaxChange: (value: string) => void;
    tip: string;
    onTipChange: (value: string) => void;
    items: ConfirmedExpenseItem[];
    onItemsChange: (items: ConfirmedExpenseItem[]) => void;
    amountStep?: string;
    validationMessage?: string | null;
    showSummaryFields?: boolean;
}

const newItem = (): ConfirmedExpenseItem => ({
    id: crypto.randomUUID(),
    description: "",
    quantity: "",
    unit: "",
    unitPrice: "",
    lineTotal: "",
});

export function ConfirmedReceiptDetails({
    merchant,
    onMerchantChange,
    subtotal,
    onSubtotalChange,
    tax,
    onTaxChange,
    tip,
    onTipChange,
    items,
    onItemsChange,
    amountStep,
    validationMessage,
    showSummaryFields = true,
}: ConfirmedReceiptDetailsProps) {
    const updateItem = (id: string, key: keyof ConfirmedExpenseItem, value: string) => {
        onItemsChange(items.map((item) => item.id === id ? { ...item, [key]: value } : item));
    };
    const moveItem = (index: number, offset: -1 | 1) => {
        const destination = index + offset;
        if (destination < 0 || destination >= items.length) return;
        const next = [...items];
        [next[index], next[destination]] = [next[destination], next[index]];
        onItemsChange(next);
    };

    return (
        <section className="mt-5 rounded-2xl border border-border bg-muted/20 p-4 md:p-5" aria-labelledby="receipt-details-title">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                    <h2 id="receipt-details-title" className="text-base font-semibold">Receipt details</h2>
                    <p className="mt-1 text-sm text-muted-foreground">Only these reviewed values will be saved with the expense.</p>
                </div>
                <Button type="button" variant="outline" className="min-h-11" onClick={() => onItemsChange([...items, newItem()])}>
                    <Icon path={mdiPlus} size={0.8} data-icon="inline-start" aria-hidden="true" />
                    Add item
                </Button>
            </div>

            {showSummaryFields ? <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <label className="grid gap-1 text-sm font-medium sm:col-span-2">
                    Merchant
                    <input className="ui-input-shell bg-background px-4" value={merchant} onChange={(event) => onMerchantChange(event.target.value)} maxLength={256} />
                </label>
                {([
                    ["Subtotal", subtotal, onSubtotalChange],
                    ["Tax", tax, onTaxChange],
                    ["Tip", tip, onTipChange],
                ] as const).map(([label, value, onChange]) => (
                    <label key={label} className="grid gap-1 text-sm font-medium">
                        {label}
                        <input className="ui-input-shell bg-background px-4" type="number" min="0" step={amountStep} value={value} onChange={(event) => onChange(event.target.value)} inputMode="decimal" />
                    </label>
                ))}
            </div> : null}

            {items.length > 0 && showSummaryFields ? <Separator className="my-5" /> : null}
            <div className="grid gap-3">
                {items.map((item, index) => (
                    <fieldset key={item.id} className="rounded-2xl border border-border bg-background p-3 md:px-4 md:py-3">
                        <legend className="px-1 text-sm font-semibold">Item {index + 1}</legend>
                        <div className="grid grid-cols-[minmax(0,1fr)_minmax(7rem,0.55fr)] items-end gap-3 md:grid-cols-12">
                            <label className="order-1 grid min-w-0 gap-1 text-sm md:col-span-4">
                                Description
                                <input className="ui-input-shell min-w-0 px-3" value={item.description} maxLength={256} onChange={(event) => updateItem(item.id, "description", event.target.value)} />
                            </label>
                            <div className="hidden md:contents">
                                <label className="grid min-w-0 gap-1 text-sm md:order-2 md:col-span-1">
                                    Quantity
                                    <input className="ui-input-shell min-w-0 px-3" inputMode="decimal" value={item.quantity} onChange={(event) => updateItem(item.id, "quantity", event.target.value)} />
                                </label>
                                <label className="grid min-w-0 gap-1 text-sm md:order-3 md:col-span-1">
                                    Unit
                                    <input className="ui-input-shell min-w-0 px-3" value={item.unit} maxLength={32} onChange={(event) => updateItem(item.id, "unit", event.target.value)} />
                                </label>
                                <label className="grid min-w-0 gap-1 text-sm md:order-4 md:col-span-2">
                                    Unit price
                                    <input className="ui-input-shell min-w-0 px-3" inputMode="decimal" value={item.unitPrice} onChange={(event) => updateItem(item.id, "unitPrice", event.target.value)} />
                                </label>
                            </div>
                            <details className="group order-3 col-span-2 rounded-xl border border-border bg-muted/20 md:hidden">
                                <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between px-3 text-sm font-medium">
                                    Quantity, unit &amp; price
                                    <span className="text-xs font-normal text-muted-foreground group-open:hidden">Show</span>
                                    <span className="hidden text-xs font-normal text-muted-foreground group-open:inline">Hide</span>
                                </summary>
                                <div className="grid grid-cols-2 gap-3 border-t border-border p-3">
                                    <label className="grid min-w-0 gap-1 text-sm">
                                        Quantity
                                        <input className="ui-input-shell min-w-0 px-3" inputMode="decimal" value={item.quantity} onChange={(event) => updateItem(item.id, "quantity", event.target.value)} />
                                    </label>
                                    <label className="grid min-w-0 gap-1 text-sm">
                                        Unit
                                        <input className="ui-input-shell min-w-0 px-3" value={item.unit} maxLength={32} onChange={(event) => updateItem(item.id, "unit", event.target.value)} />
                                    </label>
                                    <label className="col-span-2 grid min-w-0 gap-1 text-sm">
                                        Unit price
                                        <input className="ui-input-shell min-w-0 px-3" inputMode="decimal" value={item.unitPrice} onChange={(event) => updateItem(item.id, "unitPrice", event.target.value)} />
                                    </label>
                                </div>
                            </details>
                            <label className="order-2 grid min-w-0 gap-1 text-sm md:order-5 md:col-span-2">
                                Line total
                                <input className="ui-input-shell min-w-0 px-3" type="number" min="0" step={amountStep} value={item.lineTotal} onChange={(event) => updateItem(item.id, "lineTotal", event.target.value)} inputMode="decimal" />
                            </label>
                            <div className="order-4 col-span-2 flex items-end justify-end gap-2 md:order-6 md:col-span-2">
                                <Button type="button" variant="outline" size="icon" className="size-11" disabled={index === 0} aria-label={`Move item ${index + 1} up`} onClick={() => moveItem(index, -1)}><Icon path={mdiArrowUp} size={0.8} aria-hidden="true" /></Button>
                                <Button type="button" variant="outline" size="icon" className="size-11" disabled={index === items.length - 1} aria-label={`Move item ${index + 1} down`} onClick={() => moveItem(index, 1)}><Icon path={mdiArrowDown} size={0.8} aria-hidden="true" /></Button>
                                <Button type="button" variant="ghost" size="icon" className="size-11 text-destructive" aria-label={`Remove item ${index + 1}`} onClick={() => onItemsChange(items.filter(({ id }) => id !== item.id))}><Icon path={mdiDeleteOutline} size={0.85} aria-hidden="true" /></Button>
                            </div>
                        </div>
                    </fieldset>
                ))}
            </div>
            {validationMessage ? <p className="mt-4 text-sm text-destructive" role="alert">{validationMessage}</p> : null}
        </section>
    );
}
