import { useEffect, useId, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiArrowLeft, mdiCameraRetakeOutline, mdiCheck } from "@mdi/js";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel, FieldError } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmedReceiptDetails } from "@/components/expense/ConfirmedReceiptDetails";
import { ExpenseDateInput } from "@/components/expense/ExpenseDateInput";
import { useCreateExpense } from "@/hooks/CreateExpenseContextHooks";
import { editableOCRDraft } from "@/lib/receiptOcr";
import { validateReceiptExpense } from "@/lib/receiptDraft";
import { isDateOnly } from "@/lib/dateOnly";
import { moneyInputStep } from "@/lib/money";
import type { EditableOCRDraft, EditableOCRField, ReviewedReceiptDraft } from "@/types/ocr";

function ReviewCue({ field, id }: { field: EditableOCRField; id?: string }) {
    return field.requiresReview && !field.edited ? <Badge id={id} variant="secondary">Check</Badge> : null;
}

function FieldSource({ field }: { field: EditableOCRField }) {
    const source = !field.edited && field.provenance === "inferred" ? "Inferred" : null;
    return source ? <span className="text-xs text-muted-foreground">{source}</span> : null;
}

function ReviewField({
    label, field, onChange, type = "text", step,
}: {
    label: string;
    field: EditableOCRField;
    onChange: (field: EditableOCRField) => void;
    type?: "text" | "number";
    step?: string;
}) {
    const id = useId();
    const hasInferredSource = !field.edited && field.provenance === "inferred";
    return (
        <Field className="min-w-0 gap-2">
            <div className="flex min-h-5 items-center gap-2">
                <FieldLabel htmlFor={id}>{label}</FieldLabel>
                <ReviewCue field={field} id={`${id}-review`} />
            </div>
            <Input
                id={id}
                variant="expense"
                type={type}
                step={step}
                min={type === "number" ? 0 : undefined}
                inputMode={type === "number" ? "decimal" : undefined}
                value={field.value}
                aria-describedby={[hasInferredSource ? `${id}-source` : null, field.requiresReview && !field.edited ? `${id}-review` : null].filter(Boolean).join(" ") || undefined}
                onChange={(event) => onChange({ ...field, value: event.target.value, edited: true })}
            />
            {hasInferredSource ? <div id={`${id}-source`}><FieldSource field={field} /></div> : null}
        </Field>
    );
}

interface ReceiptDraftReviewProps {
    photoURL?: string | null;
    onEditPhoto?: () => void;
}

export default function ReceiptDraftReview({ photoURL, onEditPhoto }: ReceiptDraftReviewProps) {
    const navigate = useNavigate();
    const {
        selectedGroupId,
        currency,
        amountDigits,
        occurredOn,
        ocrStatus,
        ocrDraft,
        ocrError,
        applyReviewedReceipt,
        clearReceiptWorkflow,
    } = useCreateExpense();
    const [draft, setDraft] = useState<EditableOCRDraft | null>(null);
    const [detailsOpen, setDetailsOpen] = useState(false);

    useEffect(() => {
        setDraft(ocrDraft ? (() => {
            const editable = editableOCRDraft(ocrDraft);
            if (editable.date.value) return editable;
            return {
                ...editable,
                date: { value: occurredOn, provenance: "inferred", requiresReview: true, edited: false },
            };
        })() : null);
    }, [occurredOn, ocrDraft]);

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

    useEffect(() => {
        if (reviewValidation?.message) setDetailsOpen(true);
    }, [reviewValidation?.message]);

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
            <main className="page-shell receipt-review-page">
                <div className="page-container max-w-3xl">
                    <section className="panel-card grid min-h-72 place-items-center rounded-[2rem] p-8 text-center" role="status" aria-live="polite" aria-busy="true">
                        <div>
                            <Spinner className="mx-auto size-8" aria-hidden="true" />
                            {photoURL ? <img className="mx-auto mb-4 h-28 max-w-32 rounded-lg border border-border object-contain" src={photoURL} alt="Prepared receipt being scanned" /> : null}
                            <h1 className="mt-5 text-xl font-semibold">Reading your receipt</h1>
                            <p className="mt-2 text-sm text-muted-foreground">This usually takes a few seconds. No expense will be saved yet.</p>
                            <Button type="button" variant="ghost" className="mt-5 min-h-11" onClick={onEditPhoto ?? goManual}>{onEditPhoto ? "Edit photo" : "Cancel and enter manually"}</Button>
                        </div>
                    </section>
                </div>
            </main>
        );
    }

    if (ocrStatus === "error" || !draft) {
        return (
            <main className="page-shell receipt-review-page">
                <div className="page-container max-w-3xl">
                    <section className="panel-card rounded-[2rem] p-6 md:p-8" role="alert">
                        <h1 className="text-xl font-semibold">We couldn’t create a receipt draft</h1>
                        <p className="mt-2 text-sm text-muted-foreground">{ocrError ?? "The draft is no longer available."}</p>
                        <div className="mt-6 flex flex-col gap-2 sm:flex-row">
                            <Button type="button" className="min-h-11" onClick={onEditPhoto ?? (() => navigate(`../receipt${selectedGroupId ? `?g=${encodeURIComponent(selectedGroupId)}` : ""}`, { replace: true }))}><Icon path={mdiCameraRetakeOutline} size={0.85} data-icon="inline-start" aria-hidden="true" />{onEditPhoto ? "Edit and scan again" : "Try another photo"}</Button>
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
    const itemReviewNotices = Object.fromEntries(draft.items
        .filter((item) => item.quantity.provenance === "inferred" && item.quantity.requiresReview && !item.quantity.edited)
        .map((item) => [item.id, "Check inferred quantity"]));
    const confirmedItems = draft.items.map((item) => ({
        id: item.id,
        description: item.description.value,
        quantity: item.quantity.value,
        unit: item.unit.value,
        unitPrice: item.unitPrice.value,
        lineTotal: item.lineTotal.value,
    }));

    const dateIsInferred = draft.date.provenance === "inferred" && !draft.date.edited;
    const breakdownNeedsReview = [draft.subtotal, draft.tax, draft.tip, ...draft.items.flatMap((item) => [item.description, item.quantity, item.unitPrice, item.lineTotal])]
        .some((field) => field.requiresReview && !field.edited);

    return (
        <main className="page-shell receipt-review-page">
            <div className="page-container max-w-3xl">
                {!onEditPhoto ? <Button type="button" variant="ghost" className="mb-4 min-h-11" onClick={goManual}><Icon path={mdiArrowLeft} size={0.85} data-icon="inline-start" aria-hidden="true" />Back to manual entry</Button> : null}
                <header className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Receipt draft</div>
                        <h1 className="page-title">Check your receipt</h1>
                        <p className="page-copy">Confirm the key details. Nothing is saved yet.</p>
                    </div>
                </header>

                <div className="grid gap-4">
                    {photoURL ? (
                        <div className="flex items-center gap-3 rounded-2xl border border-border bg-card p-3">
                            <img className="h-20 w-16 shrink-0 rounded-lg border border-border object-contain" src={photoURL} alt="Prepared receipt sent for scanning" />
                            <div className="min-w-0 flex-1">
                                <p className="text-sm font-medium">Your scanned photo</p>
                                <p className="text-xs text-muted-foreground">Cropped and masked version</p>
                            </div>
                            {onEditPhoto ? <Button type="button" variant="outline" className="min-h-11 shrink-0" onClick={onEditPhoto}>Edit photo</Button> : null}
                        </div>
                    ) : null}
                    <section className="panel-card rounded-2xl p-4 sm:p-6" aria-labelledby="summary-title">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                            <h2 id="summary-title" className="text-lg font-semibold">Key details</h2>
                            <Badge variant="secondary">{currency}</Badge>
                        </div>
                        <FieldGroup className="mt-5 grid items-start gap-4 sm:grid-cols-2">
                            <ReviewField label="Merchant" field={draft.merchant} onChange={(field) => updateSummary("merchant", field)} />
                            <Field className="min-w-0 gap-2" data-invalid={!dateValid}>
                                <div className="flex min-h-5 items-center gap-2">
                                    <FieldLabel htmlFor="receipt-date">Date</FieldLabel>
                                    <ReviewCue field={draft.date} id="receipt-date-review" />
                                </div>
                                <ExpenseDateInput
                                    id="receipt-date"
                                    value={draft.date.value}
                                    onChange={(event) => updateSummary("date", { ...draft.date, value: event.target.value, edited: true })}
                                    aria-invalid={!dateValid}
                                    aria-describedby={[dateIsInferred ? "receipt-date-source" : null, draft.date.requiresReview && !draft.date.edited ? "receipt-date-review" : null, !dateValid ? "receipt-date-error" : null].filter(Boolean).join(" ") || undefined}
                                />
                                {dateIsInferred ? <div id="receipt-date-source" className="grid gap-1">
                                    <FieldSource field={draft.date} />
                                    <p className="text-xs text-muted-foreground">No receipt date was found. Confirm the selected expense date.</p>
                                </div> : null}
                                {!dateValid ? <FieldError id="receipt-date-error">Choose a valid date.</FieldError> : null}
                            </Field>
                            <div className="sm:col-span-2">
                                <ReviewField label="Total" field={draft.total} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("total", field)} />
                            </div>
                        </FieldGroup>
                        {draft.currencySuggestion.value && draft.currencySuggestion.value.toUpperCase() !== currency.toUpperCase() ? (
                            <p className="mt-3 text-sm text-destructive">Receipt says {draft.currencySuggestion.value.toUpperCase()}, but this expense uses {currency}. Check the amounts.</p>
                        ) : null}
                    </section>

                    <details className="panel-card rounded-2xl p-4 sm:p-6" open={detailsOpen} onToggle={(event) => setDetailsOpen(event.currentTarget.open)}>
                        <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
                            <span className="min-w-0">
                                <span className="block text-lg font-semibold">Breakdown</span>
                                <span className="block text-sm text-muted-foreground">{draft.items.length} item{draft.items.length === 1 ? "" : "s"} · subtotal, tax &amp; tip</span>
                            </span>
                            <span className="flex shrink-0 items-center gap-2">
                                {breakdownNeedsReview || reviewValidation?.message ? <Badge variant="secondary">Check details</Badge> : null}
                                <span aria-hidden="true">{detailsOpen ? "−" : "+"}</span>
                            </span>
                        </summary>
                        <div className="mt-4 grid gap-4">
                            <section aria-labelledby="amounts-title">
                                <h3 id="amounts-title" className="text-sm font-semibold">Amounts</h3>
                                <FieldGroup className="mt-3 grid grid-cols-2 items-start gap-4">
                                    <ReviewField label="Subtotal" field={draft.subtotal} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("subtotal", field)} />
                                    <ReviewField label="Tax" field={draft.tax} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("tax", field)} />
                                    <ReviewField label="Tip" field={draft.tip} type="number" step={amountDigits === null ? undefined : moneyInputStep(amountDigits)} onChange={(field) => updateSummary("tip", field)} />
                                </FieldGroup>
                                {reviewValidation?.message && reviewValidation.section !== "items" ? <FieldError className="mt-4">{reviewValidation.message}</FieldError> : null}
                            </section>
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
                                itemReviewNotices={itemReviewNotices}
                                validationMessage={reviewValidation?.section === "items" ? reviewValidation.message : null}
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
                                showSummaryFields={false}
                            />
                        </div>
                    </details>
                    <div className="receipt-workspace-actions flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                        <p className="hidden text-sm text-muted-foreground sm:block">You’ll choose who paid and how to split next.</p>
                        <Button type="button" className="min-h-12 w-full sm:w-auto" disabled={!reviewValidation?.valid || !dateValid} onClick={apply}><Icon path={mdiCheck} size={0.85} data-icon="inline-start" aria-hidden="true" />Use reviewed values</Button>
                    </div>
                </div>
            </div>
        </main>
    );
}
