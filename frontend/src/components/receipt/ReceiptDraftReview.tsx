import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiArrowLeft, mdiCameraRetakeOutline, mdiCheck, mdiReceiptTextEditOutline } from "@mdi/js";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmedReceiptDetails } from "@/components/expense/ConfirmedReceiptDetails";
import { useCreateExpense } from "@/hooks/CreateExpenseContextHooks";
import { editableOCRDraft } from "@/lib/receiptOcr";
import { validateReceiptExpense } from "@/lib/receiptDraft";
import { isDateOnly } from "@/lib/dateOnly";
import { moneyInputStep } from "@/lib/money";
import type { EditableOCRDraft, EditableOCRField, ReviewedReceiptDraft } from "@/types/ocr";

function fieldBadge(field: EditableOCRField) {
    if (field.edited) return <Badge variant="outline">Edited</Badge>;
    if (field.provenance === "inferred") return <Badge variant="secondary">Inferred</Badge>;
    if (field.provenance === "provider") return <Badge variant="outline">Extracted</Badge>;
    return null;
}

function ReviewField({
    label,
    field,
    onChange,
    type = "text",
    requiredReview = false,
    step,
}: {
    label: string;
    field: EditableOCRField;
    onChange: (field: EditableOCRField) => void;
    type?: "text" | "number";
    requiredReview?: boolean;
    step?: string;
}) {
    return (
        <label className="grid gap-1.5 text-sm font-medium">
            <span className="flex min-h-6 flex-wrap items-center gap-2">
                {label}
                {fieldBadge(field)}
                {requiredReview ? <Badge variant="secondary">Check this</Badge> : null}
            </span>
            <input
                className="ui-input-shell bg-background px-4"
                type={type}
                step={step}
                min={type === "number" ? 0 : undefined}
                inputMode={type === "number" ? "decimal" : undefined}
                value={field.value}
                onChange={(event) => onChange({ ...field, value: event.target.value, edited: true })}
            />
            {typeof field.confidence === "number" && !field.edited ? (
                <span className="text-xs font-normal text-muted-foreground">Extraction confidence {Math.round(field.confidence)}%</span>
            ) : null}
        </label>
    );
}

export default function ReceiptDraftReview() {
    const navigate = useNavigate();
    const {
        selectedGroupId,
        currency,
        amountDigits,
        ocrStatus,
        ocrDraft,
        ocrError,
        applyReviewedReceipt,
        clearReceiptWorkflow,
    } = useCreateExpense();
    const [draft, setDraft] = useState<EditableOCRDraft | null>(null);

    useEffect(() => {
        setDraft(ocrDraft ? editableOCRDraft(ocrDraft) : null);
    }, [ocrDraft]);

    const returnTo = `/create_expense${selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : ""}`;
    const reviewValidation = useMemo(() => draft ? validateReceiptExpense({
        merchant: draft.merchant.value,
        subtotal: draft.subtotal.value,
        tax: draft.tax.value,
        tip: draft.tip.value,
        total: draft.total.value,
        items: draft.items.map((item) => ({
            id: item.id,
            description: item.description.value,
            quantity: item.quantity.value,
            unit: item.unit.value,
            unitPrice: item.unitPrice.value,
            lineTotal: item.lineTotal.value,
        })),
    }, amountDigits) : null, [amountDigits, draft]);
    const dateValid = Boolean(draft && isDateOnly(draft.date.value));

    const goManual = () => {
        clearReceiptWorkflow();
        navigate(returnTo, { replace: true });
    };

    const apply = () => {
        if (!draft || !reviewValidation?.valid || !dateValid) return;
        const reviewed: ReviewedReceiptDraft = {
            merchant: draft.merchant.value.trim(),
            date: draft.date.value,
            currencySuggestion: draft.currencySuggestion.value.trim(),
            subtotal: draft.subtotal.value,
            tax: draft.tax.value,
            tip: draft.tip.value,
            total: draft.total.value,
            items: draft.items.map((item) => ({
                id: item.id,
                description: item.description.value,
                quantity: item.quantity.value,
                unit: item.unit.value,
                unitPrice: item.unitPrice.value,
                lineTotal: item.lineTotal.value,
            })),
        };
        applyReviewedReceipt(reviewed);
        navigate(returnTo, { replace: true });
    };

    if (ocrStatus === "scanning") {
        return (
            <main className="page-shell">
                <div className="page-container max-w-3xl">
                    <section className="panel-card grid min-h-72 place-items-center rounded-[2rem] p-8 text-center" role="status" aria-live="polite" aria-busy="true">
                        <div>
                            <Spinner className="mx-auto size-8" aria-hidden="true" />
                            <h1 className="mt-5 text-xl font-semibold">Reading your receipt</h1>
                            <p className="mt-2 text-sm text-muted-foreground">This usually takes a few seconds. No expense will be saved yet.</p>
                            <Button type="button" variant="ghost" className="mt-5 min-h-11" onClick={goManual}>Cancel and enter manually</Button>
                        </div>
                    </section>
                </div>
            </main>
        );
    }

    if (ocrStatus === "error" || !draft) {
        return (
            <main className="page-shell">
                <div className="page-container max-w-3xl">
                    <section className="panel-card rounded-[2rem] p-6 md:p-8" role="alert">
                        <h1 className="text-xl font-semibold">We couldn’t create a receipt draft</h1>
                        <p className="mt-2 text-sm text-muted-foreground">{ocrError ?? "The draft is no longer available."}</p>
                        <div className="mt-6 flex flex-col gap-2 sm:flex-row">
                            <Button type="button" className="min-h-11" onClick={() => navigate(`../receipt${selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : ""}`, { replace: true })}><Icon path={mdiCameraRetakeOutline} size={0.85} data-icon="inline-start" aria-hidden="true" />Try another photo</Button>
                            <Button type="button" variant="outline" className="min-h-11" onClick={goManual}>Continue manually</Button>
                        </div>
                    </section>
                </div>
            </main>
        );
    }

    const updateSummary = (key: keyof Pick<EditableOCRDraft, "merchant" | "date" | "currencySuggestion" | "subtotal" | "tax" | "tip" | "total">, field: EditableOCRField) => {
        setDraft((current) => current ? { ...current, [key]: field } : current);
    };
    const confirmedItems = draft.items.map((item) => ({
        id: item.id,
        description: item.description.value,
        quantity: item.quantity.value,
        unit: item.unit.value,
        unitPrice: item.unitPrice.value,
        lineTotal: item.lineTotal.value,
    }));

    return (
        <main className="page-shell">
            <div className="page-container max-w-5xl">
                <Button type="button" variant="ghost" className="mb-4 min-h-11" onClick={goManual}><Icon path={mdiArrowLeft} size={0.85} data-icon="inline-start" aria-hidden="true" />Back to manual entry</Button>
                <header className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Receipt draft</div>
                        <h1 className="page-title">Review every value</h1>
                        <p className="page-copy">OCR suggestions can be wrong. Nothing is saved until you return to the expense form and press Save Expense.</p>
                    </div>
                </header>

                <section className="panel-card rounded-[2rem] p-4 sm:p-6 md:p-8" aria-labelledby="summary-title">
                    <div className="flex items-center gap-3">
                        <span className="grid size-11 place-items-center rounded-xl bg-primary/10 text-primary" aria-hidden="true"><Icon path={mdiReceiptTextEditOutline} size={1} /></span>
                        <div><h2 id="summary-title" className="text-lg font-semibold">Receipt summary</h2><p className="text-sm text-muted-foreground">Merchant and date always need your review.</p></div>
                    </div>
                    <div className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                        <ReviewField label="Merchant" field={draft.merchant} requiredReview onChange={(field) => updateSummary("merchant", field)} />
                        <ReviewField label="Date (YYYY-MM-DD)" field={draft.date} requiredReview onChange={(field) => updateSummary("date", field)} />
                        <ReviewField label="OCR currency suggestion" field={draft.currencySuggestion} onChange={(field) => updateSummary("currencySuggestion", field)} />
                        <ReviewField label="Total" field={draft.total} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("total", field)} />
                        <ReviewField label="Subtotal" field={draft.subtotal} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("subtotal", field)} />
                        <ReviewField label="Tax" field={draft.tax} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("tax", field)} />
                        <ReviewField label="Tip" field={draft.tip} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("tip", field)} />
                    </div>
                    <div className="mt-4 rounded-xl border border-border bg-muted/40 px-4 py-3 text-sm">
                        <strong>{currency}</strong> is the selected group currency and will remain authoritative.
                        {draft.currencySuggestion.value && draft.currencySuggestion.value.toUpperCase() !== currency.toUpperCase() ? <span className="mt-1 block text-amber-700 dark:text-amber-300">The receipt suggested {draft.currencySuggestion.value}; check the amounts before continuing.</span> : null}
                    </div>
                    {!dateValid ? <p className="mt-3 text-sm text-destructive" role="alert">Enter the date as YYYY-MM-DD.</p> : null}

                    <ConfirmedReceiptDetails
                        merchant={draft.merchant.value}
                        onMerchantChange={(value) => updateSummary("merchant", { ...draft.merchant, value, edited: true })}
                        subtotal={draft.subtotal.value}
                        onSubtotalChange={(value) => updateSummary("subtotal", { ...draft.subtotal, value, edited: true })}
                        tax={draft.tax.value}
                        onTaxChange={(value) => updateSummary("tax", { ...draft.tax, value, edited: true })}
                        tip={draft.tip.value}
                        onTipChange={(value) => updateSummary("tip", { ...draft.tip, value, edited: true })}
                        items={confirmedItems}
                        onItemsChange={(items) => setDraft((current) => current ? {
                            ...current,
                            items: items.map((item) => {
                                const prior = current.items.find(({ id }) => id === item.id);
                                const changed = (value: string, old?: EditableOCRField): EditableOCRField => old && old.value === value ? old : { ...(old ?? { value }), value, edited: true };
                                return {
                                    id: item.id,
                                    description: changed(item.description, prior?.description),
                                    quantity: changed(item.quantity, prior?.quantity),
                                    unit: changed(item.unit, prior?.unit),
                                    unitPrice: changed(item.unitPrice, prior?.unitPrice),
                                    lineTotal: changed(item.lineTotal, prior?.lineTotal),
                                };
                            }),
                        } : current)}
                        amountStep={amountDigits === null ? undefined : moneyInputStep(amountDigits)}
                        validationMessage={reviewValidation?.message}
                        showSummaryFields={false}
                    />
                    <div className="mt-6 flex flex-col-reverse gap-3 border-t border-border pt-5 sm:flex-row sm:items-center sm:justify-between">
                        <Button type="button" variant="ghost" className="min-h-11" onClick={goManual}>Discard draft</Button>
                        <Button type="button" className="min-h-12" disabled={!reviewValidation?.valid || !dateValid} onClick={apply}><Icon path={mdiCheck} size={0.85} data-icon="inline-start" aria-hidden="true" />Use reviewed values</Button>
                    </div>
                </section>
            </div>
        </main>
    );
}
