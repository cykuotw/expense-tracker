import { useId, useLayoutEffect, useRef, useState } from "react";
import type { ExpenseCategoryAmount, ExpenseTypeItem } from "../../types/expense";
import { calculateCategoryBreakdown } from "../../lib/categoryBreakdown";
import { decimalToUnits, unitsToDecimal } from "../../lib/money";
import { ExpenseTypePicker } from "./ExpenseTypePicker";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "../ui/field";

interface Props {
    rows: (ExpenseCategoryAmount & { clientId?: string })[];
    onChange: (rows: ExpenseCategoryAmount[]) => void;
    expenseTypes: ExpenseTypeItem[];
    singleTypeId: string;
    onSingleTypeChange: (id: string) => void;
    total: string;
    currency: string;
    amountDigits: number | null;
}

export function ExpenseCategoryEditor({ rows, onChange, expenseTypes, singleTypeId, onSingleTypeChange, total, currency, amountDigits }: Props) {
    const id = useId();
    const root = useRef<HTMLFieldSetElement>(null);
    const pendingFocus = useRef<string | null>(null);
    const selectFocusedAmount = useRef(false);
    const [touched, setTouched] = useState<number[]>([]);
    const [noteRows, setNoteRows] = useState<number[]>([]);
    const [converting, setConverting] = useState(false);
    const [retainedType, setRetainedType] = useState("");
    const calculation = calculateCategoryBreakdown(rows, total, amountDigits);
    const split = rows.length > 0;
    const money = (units: bigint) => `${currency} ${unitsToDecimal(units, amountDigits ?? 2)}`;
    const availableTypes = (index: number) => expenseTypes.filter(type => type.id === rows[index]?.expenseTypeId || !rows.some(row => row.expenseTypeId === type.id));
    const update = (index: number, patch: Partial<ExpenseCategoryAmount>) => onChange(rows.map((row, i) => i === index ? { ...row, ...patch } : row));
    const focusAmount = (index: number, select = false) => {
        pendingFocus.current = `[data-category-amount="${index}"]`;
        selectFocusedAmount.current = select;
    };
    useLayoutEffect(() => {
        if (pendingFocus.current === null) return;
        const target = root.current?.querySelector<HTMLElement>(pendingFocus.current);
        const select = selectFocusedAmount.current;
        pendingFocus.current = null;
        selectFocusedAmount.current = false;
        target?.focus();
        if (select && target instanceof HTMLInputElement) target.select();
    });
    const begin = () => {
        const initialRows = [{ clientId: crypto.randomUUID(), expenseTypeId: singleTypeId, amount: total }, { clientId: crypto.randomUUID(), expenseTypeId: "", amount: "" }];
        onChange(initialRows);
        setTouched([]);
        setNoteRows([]);
        focusAmount(0, true);
    };
    const finishConversion = (type: string) => {
        onSingleTypeChange(type);
        onChange([]);
        setConverting(false);
        setTouched([]);
        setNoteRows([]);
        pendingFocus.current = "[data-split-categories]";
    };
    const remove = (index: number) => {
        if (rows.length === 2) {
            const other = rows[1 - index];
            const totalUnits = amountDigits === null ? null : decimalToUnits(total, amountDigits);
            const retainedUnits = amountDigits === null ? null : decimalToUnits(other.amount, amountDigits);
            const removedRowIsEmpty = !rows[index].expenseTypeId && !rows[index].amount && !rows[index].note;
            const retainedRowIsFullExpense = Boolean(other.expenseTypeId) && !other.note &&
                totalUnits !== null && totalUnits > 0n && retainedUnits === totalUnits;
            if (removedRowIsEmpty && retainedRowIsFullExpense) finishConversion(other.expenseTypeId);
            else { setRetainedType(other.expenseTypeId); pendingFocus.current = "[data-conversion] button"; setConverting(true); }
            return;
        }
        onChange(rows.filter((_, i) => i !== index));
        setTouched([]);
        setNoteRows(noteRows.filter(i => i !== index).map(i => i > index ? i - 1 : i));
        focusAmount(Math.min(index, rows.length - 2));
    };

    return (
        <FieldSet ref={root} className="mt-5" aria-describedby={`${id}-help`}>
            <FieldLegend>Category</FieldLegend>
            {!split ? (
                <FieldGroup>
                    <Field>
                        <ExpenseTypePicker expenseTypes={expenseTypes} value={singleTypeId} onChange={onSingleTypeChange} />
                        <Button data-split-categories type="button" variant="outline" className="min-h-11 self-start" onClick={begin}>Split across categories</Button>
                    </Field>
                    <FieldDescription id={`${id}-help`}>Choose what this expense was for.</FieldDescription>
                </FieldGroup>
            ) : (
                <FieldGroup>
                    <FieldDescription id={`${id}-help`}>Bought different things? Give each category part of the total. This stays one expense.</FieldDescription>
                    <div className="flex flex-wrap justify-between gap-2 text-sm">
                        <span>Expense total: <strong>{currency} {total || "—"}</strong></span>
                        <span>Include tax and fees in each amount.</span>
                    </div>
                    {rows.map((row, index) => {
                        const amount = amountDigits === null ? null : decimalToUnits(row.amount, amountDigits);
                        const invalidAmount = touched.includes(index) && (amount === null || amount <= 0n);
                        const missingCategory = touched.includes(index) && !row.expenseTypeId;
                        const categoryErrorId = `${id}-${index}-category-error`;
                        const amountErrorId = `${id}-${index}-amount-error`;
                        return (
                            <FieldSet key={row.clientId ?? index} className="gap-3 rounded-2xl border border-border p-3 sm:p-4">
                                <FieldLegend variant="label">Category {index + 1}</FieldLegend>
                                <FieldGroup className="gap-3">
                                    <Field data-invalid={missingCategory || undefined}>
                                        <ExpenseTypePicker expenseTypes={availableTypes(index)} value={row.expenseTypeId}
                                            label={`Category ${index + 1}: ${expenseTypes.find(type => type.id === row.expenseTypeId)?.name ?? "Choose a category"}`}
                                            invalid={missingCategory} describedBy={missingCategory ? categoryErrorId : undefined}
                                            onChange={(expenseTypeId) => update(index, { expenseTypeId })} />
                                    </Field>
                                    <Field data-invalid={invalidAmount || undefined}>
                                        <FieldLabel htmlFor={`${id}-${index}-amount`}>Amount ({currency})</FieldLabel>
                                        <Input id={`${id}-${index}-amount`} data-category-amount={index} variant="expense" className="min-h-11"
                                            type="text" inputMode="decimal" value={row.amount} required disabled={amountDigits === null}
                                            aria-invalid={invalidAmount || undefined} aria-describedby={invalidAmount ? amountErrorId : undefined}
                                            aria-label={`Amount for category ${index + 1} (${currency})`}
                                            onChange={event => update(index, { amount: event.target.value })}
                                            onBlur={() => setTouched(current => current.includes(index) ? current : [...current, index])} />
                                        {row.amount === "" && calculation.remaining !== null && calculation.remaining > 0n ? (
                                            <Button type="button" variant="outline" className="min-h-11 self-start" onClick={() => { update(index, { amount: unitsToDecimal(calculation.remaining!, amountDigits!) }); focusAmount(index); }}>
                                                Use remaining {money(calculation.remaining)}
                                            </Button>
                                        ) : null}
                                    </Field>
                                    {missingCategory ? <p id={categoryErrorId} className="text-sm text-destructive">Choose a category.</p> : null}
                                    {invalidAmount ? <p id={amountErrorId} className="text-sm text-destructive">Enter an amount greater than zero with up to {amountDigits ?? 0} decimal places.</p> : null}
                                    {noteRows.includes(index) || row.note ? (
                                        <Field>
                                            <FieldLabel htmlFor={`${id}-${index}-note`}>Note (optional)</FieldLabel>
                                            <Input id={`${id}-${index}-note`} data-category-note={index} className="min-h-11" value={row.note ?? ""} maxLength={200}
                                                onChange={event => update(index, { note: event.target.value })} />
                                        </Field>
                                    ) : null}
                                    {/* Prevent blur validation from moving actions before a pointer click completes. */}
                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                        {!noteRows.includes(index) && !row.note ? <Button type="button" variant="ghost" className="min-h-11" onPointerDown={event => event.preventDefault()} aria-label={`Add note to category ${index + 1}`} onClick={() => { setNoteRows(current => [...current, index]); pendingFocus.current = `[data-category-note="${index}"]`; }}>Add note<span className="sr-only"> to category {index + 1}</span></Button> : null}
                                        <Button type="button" variant="ghost" className="min-h-11" onPointerDown={event => event.preventDefault()} aria-label={`Remove category ${index + 1}`} onClick={() => remove(index)}>Remove<span className="sr-only"> category {index + 1}</span></Button>
                                    </div>
                                </FieldGroup>
                            </FieldSet>
                        );
                    })}
                    <div role="status" aria-label="Category amount status" aria-live="polite" aria-atomic="true" className="flex flex-col gap-1 text-sm">
                        <strong>{calculation.remaining === null ? "Enter the expense total to check category amounts." : calculation.remaining > 0n ? `${money(calculation.remaining)} left to categorize` : calculation.remaining < 0n ? `${money(-calculation.remaining)} over the total` : calculation.valid ? `All ${money(decimalToUnits(total, amountDigits!) ?? 0n)} categorized` : "Amounts match the total. Check the category entries."}</strong>
                        {calculation.error ? <span>{calculation.error}</span> : null}
                    </div>
                    <div className="flex flex-wrap gap-2">
                        <Button type="button" variant="outline" className="min-h-11" onPointerDown={event => event.preventDefault()} disabled={rows.length >= Math.min(20, expenseTypes.length)} onClick={() => { const addedRows = [...rows, { clientId: crypto.randomUUID(), expenseTypeId: "", amount: "" }]; onChange(addedRows); focusAmount(rows.length); }}>Add category</Button>
                        <Button type="button" variant="ghost" onPointerDown={event => event.preventDefault()} data-use-one-category className="min-h-11" onClick={() => { setRetainedType(rows[0]?.expenseTypeId ?? ""); pendingFocus.current = "[data-conversion] button"; setConverting(true); }}>Use one category</Button>
                    </div>
                    {rows.length === 20 ? <FieldDescription>You can use up to 20 categories.</FieldDescription> : rows.length >= expenseTypes.length ? <FieldDescription>All available categories are already included.</FieldDescription> : null}
                    {converting ? (
                        <FieldSet data-conversion className="rounded-xl border border-border p-4">
                            <FieldLegend variant="label">Use one category</FieldLegend>
                            <FieldDescription>This removes the category amounts and notes. Choose one category for the full {currency} {total}.</FieldDescription>
                            <ExpenseTypePicker expenseTypes={expenseTypes} value={retainedType} label="Category for the full expense" onChange={setRetainedType} />
                            <div className="flex flex-wrap gap-2">
                                <Button type="button" className="min-h-11" disabled={!retainedType} onClick={() => finishConversion(retainedType)}>Use this category for the full total</Button>
                                <Button type="button" variant="outline" className="min-h-11" onClick={() => { pendingFocus.current = "[data-use-one-category]"; setConverting(false); }}>Keep category amounts</Button>
                            </div>
                        </FieldSet>
                    ) : null}
                </FieldGroup>
            )}
        </FieldSet>
    );
}
