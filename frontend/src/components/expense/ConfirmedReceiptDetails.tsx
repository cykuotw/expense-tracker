import { useId, useState } from "react";
import Icon from "@mdi/react";
import { mdiArrowDown, mdiArrowUp, mdiDeleteOutline, mdiPlus, mdiChevronDown } from "@mdi/js";

import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
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
    itemReviewNotices?: Record<string, string>;
    showSummaryFields?: boolean;
    compactItems?: boolean;
}

const newItem = (): ConfirmedExpenseItem => ({
    id: crypto.randomUUID(),
    description: "",
    quantity: "",
    unit: "",
    unitPrice: "",
    lineTotal: "",
});

function ItemField({ label, value, onChange, numeric = false, type = "text", step, maxLength }: {
    label: string;
    value: string;
    onChange: (value: string) => void;
    numeric?: boolean;
    type?: "text" | "number";
    step?: string;
    maxLength?: number;
}) {
    const id = useId();
    return (
        <Field className="min-w-0 gap-2">
            <FieldLabel htmlFor={id}>{label}</FieldLabel>
            <Input id={id} variant="expense" value={value} onChange={(event) => onChange(event.target.value)}
                type={type} min={type === "number" ? 0 : undefined} step={step}
                inputMode={numeric ? "decimal" : undefined} maxLength={maxLength} />
        </Field>
    );
}

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
    itemReviewNotices,
    showSummaryFields = true,
    compactItems = false,
}: ConfirmedReceiptDetailsProps) {
    const [openItemId, setOpenItemId] = useState<string | null>(null);
    const addItem = () => {
        const item = newItem();
        onItemsChange([...items, item]);
        if (compactItems) setOpenItemId(item.id);
    };
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
        <section className={cn("receipt-details rounded-2xl p-4 sm:p-6", showSummaryFields ? "mt-5 border border-border bg-muted/20" : "panel-card")} aria-labelledby="receipt-details-title">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <h2 id="receipt-details-title" className="text-lg font-semibold">{showSummaryFields ? "Receipt details" : <>Items <span className="text-sm font-normal text-muted-foreground">({items.length})</span></>}</h2>
                {!showSummaryFields ? <Button type="button" variant="outline" className="min-h-11" onClick={addItem}>
                    <Icon path={mdiPlus} size={0.8} data-icon="inline-start" aria-hidden="true" />Add item
                </Button> : null}
            </div>

            {showSummaryFields ? <FieldGroup className="mt-4 gap-4">
                <ItemField label="Merchant" value={merchant} onChange={onMerchantChange} maxLength={256} />
                <FieldGroup className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                    {([
                        ["Subtotal", subtotal, onSubtotalChange],
                        ["Tax", tax, onTaxChange],
                        ["Tip", tip, onTipChange],
                    ] as const).map(([label, value, onChange]) => (
                        <ItemField key={label} label={label} value={value} onChange={onChange} numeric type="number" step={amountStep} />
                    ))}
                </FieldGroup>
                <Separator />
            </FieldGroup> : null}

            {showSummaryFields ? <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
                <h3 className="text-sm font-semibold">Items <span className="font-normal text-muted-foreground">({items.length})</span></h3>
                <Button type="button" variant="outline" className="min-h-11" onClick={addItem}>
                    <Icon path={mdiPlus} size={0.8} data-icon="inline-start" aria-hidden="true" />Add item
                </Button>
            </div> : null}
            {validationMessage ? <p className="mt-4 text-sm text-destructive" role="alert">{validationMessage}</p> : null}
            <div className="mt-4 grid gap-4">
                {items.map((item, index) => {
                    const isOpen = !compactItems || openItemId === item.id;
                    return <fieldset key={item.id} className={cn("min-w-0 rounded-xl border border-border", compactItems ? "p-2 sm:p-3" : "p-3 sm:p-4")}>
                        <legend className={cn("px-1 text-sm font-semibold", compactItems && "sr-only")}>Item {index + 1}</legend>
                        {compactItems ? <button
                            type="button"
                            className="flex min-h-11 w-full min-w-0 items-center gap-3 rounded-lg px-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                            aria-expanded={isOpen}
                            aria-label={`Item ${index + 1}: ${item.description.trim() || "Add description"}, ${item.lineTotal.trim() || "Add total"}`}
                            onClick={() => setOpenItemId(isOpen ? null : item.id)}
                        >
                            <span className="w-5 shrink-0 text-xs text-muted-foreground">{index + 1}</span>
                            <span className="min-w-0 flex-1 truncate text-sm font-medium">{item.description.trim() || "Add description"}</span>
                            <span className="max-w-28 shrink-0 truncate text-right text-sm tabular-nums text-muted-foreground">{item.lineTotal.trim() || "Add total"}</span>
                            <Icon path={mdiChevronDown} size={0.75} className={cn("shrink-0 transition-transform", isOpen && "rotate-180")} aria-hidden="true" />
                        </button> : null}
                        {isOpen ? <div className={cn(compactItems && "px-2 pb-1 pt-3")}>
                            <FieldGroup className="grid grid-cols-[minmax(0,1fr)_6.5rem] items-start gap-3 sm:grid-cols-[minmax(0,1fr)_10rem]">
                                <ItemField label="Description" value={item.description} onChange={(value) => updateItem(item.id, "description", value)} maxLength={256} />
                                <ItemField label="Line total" value={item.lineTotal} onChange={(value) => updateItem(item.id, "lineTotal", value)} numeric type="number" step={amountStep} />
                            </FieldGroup>
                            <details className="group mt-2">
                                <summary className="flex min-h-11 cursor-pointer list-none flex-wrap items-center justify-between gap-2 rounded-lg text-sm text-muted-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
                                    <span className="flex flex-wrap items-center gap-2">
                                        Quantity, unit &amp; price
                                        {itemReviewNotices?.[item.id] ? <span className="text-xs font-medium text-foreground">{itemReviewNotices[item.id]}</span> : null}
                                    </span>
                                    <span className="group-open:hidden" aria-hidden="true">+</span>
                                    <span className="hidden group-open:inline" aria-hidden="true">−</span>
                                </summary>
                                <FieldGroup className="grid gap-3 pb-3 sm:grid-cols-3">
                                    <ItemField label="Quantity" value={item.quantity} onChange={(value) => updateItem(item.id, "quantity", value)} numeric />
                                    <ItemField label="Unit" value={item.unit} onChange={(value) => updateItem(item.id, "unit", value)} maxLength={32} />
                                    <ItemField label="Unit price" value={item.unitPrice} onChange={(value) => updateItem(item.id, "unitPrice", value)} numeric />
                                </FieldGroup>
                            </details>
                            <div className="flex items-center justify-end gap-2" role="group" aria-label={`Item ${index + 1} actions`}>
                                <Button type="button" variant="ghost" size="icon" className="size-11" disabled={index === 0} aria-label={`Move item ${index + 1} up`} onClick={() => moveItem(index, -1)}><Icon path={mdiArrowUp} size={0.8} aria-hidden="true" /></Button>
                                <Button type="button" variant="ghost" size="icon" className="size-11" disabled={index === items.length - 1} aria-label={`Move item ${index + 1} down`} onClick={() => moveItem(index, 1)}><Icon path={mdiArrowDown} size={0.8} aria-hidden="true" /></Button>
                                <Button type="button" variant="ghost" size="icon" className="size-11" aria-label={`Remove item ${index + 1}`} onClick={() => onItemsChange(items.filter(({ id }) => id !== item.id))}><Icon path={mdiDeleteOutline} size={0.85} aria-hidden="true" /></Button>
                            </div>
                        </div> : null}
                    </fieldset>;
                })}
            </div>
            {items.length === 0 ? <p className="mt-3 text-sm text-muted-foreground">No items yet. Add an item to include a breakdown.</p> : null}
        </section>
    );
}
