import { Fragment, useCallback, useRef, useState } from "react";
import { mdiCheck, mdiChevronDown } from "@mdi/js";
import Icon from "@mdi/react";
import { CurrencyMetadata } from "../../lib/money";

interface CurrencyPickerProps {
    value: string;
    currencies: CurrencyMetadata[];
    onChange: (value: string) => void;
    disabled?: boolean;
}

const mainCurrencyCodes = new Set(["CAD", "TWD", "USD"]);

export function CurrencyPicker({
    value,
    currencies,
    onChange,
    disabled = false,
}: CurrencyPickerProps) {
    const [open, setOpen] = useState(false);
    const pointerSelectionRef = useRef<string | null>(null);
    const pointerStartYRef = useRef<number | null>(null);
    const pointerMovedRef = useRef(false);
    const otherCurrenciesStart = currencies.findIndex(
        (currency) => !mainCurrencyCodes.has(currency.code)
    );
    const selected = currencies.find((currency) => currency.code === value);
    const selectOption = useCallback(
        (nextValue: string) => {
            onChange(nextValue);
            setOpen(false);
        },
        [onChange]
    );

    return (
        <div
            className="relative"
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
            }}
            onKeyDown={(event) => {
                if (event.key === "Escape") setOpen(false);
            }}
        >
            <button
                type="button"
                aria-label="Currency"
                aria-expanded={open}
                aria-haspopup="listbox"
                disabled={disabled}
                className="flex min-h-14 w-full touch-manipulation items-center gap-3 rounded-2xl border border-border bg-background px-4 py-2 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:cursor-not-allowed disabled:opacity-60 md:min-h-20 md:py-3"
                onClick={() => {
                    pointerSelectionRef.current = null;
                    setOpen((current) => !current);
                }}
            >
                <span className="min-w-0 flex-1">
                    <span className="block truncate font-semibold">
                        {selected ? `${selected.code} — ${selected.displayName}` : "Select currency"}
                    </span>
                </span>
                <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
            </button>
            {open && (
                <div
                    role="listbox"
                    aria-label="Currency options"
                    className="absolute z-[60] mt-2 max-h-48 w-full touch-pan-y overflow-y-auto overscroll-contain rounded-2xl border border-border bg-background p-2 shadow-xl md:max-h-80"
                >
                    {currencies.map((currency, index) => (
                        <Fragment key={currency.code}>
                            {index === otherCurrenciesStart && index > 0 ? (
                                <div
                                    role="separator"
                                    aria-label="Other currencies"
                                    className="mx-3 my-1.5 border-t border-border pt-2 text-xs font-semibold uppercase tracking-[0.16em] text-foreground/50"
                                >
                                    Other currencies
                                </div>
                            ) : null}
                            <button
                            type="button"
                            role="option"
                            aria-selected={currency.code === value}
                            className={`flex min-h-12 w-full touch-manipulation items-center gap-3 rounded-xl px-3 py-2.5 text-left hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${currency.code === value ? "bg-primary/10" : ""}`}
                            onPointerDown={(event) => {
                                pointerStartYRef.current = event.clientY;
                                pointerMovedRef.current = false;
                            }}
                            onPointerMove={(event) => {
                                if (pointerStartYRef.current !== null && Math.abs(event.clientY - pointerStartYRef.current) > 8) {
                                    pointerMovedRef.current = true;
                                }
                            }}
                            onPointerUp={() => {
                                pointerStartYRef.current = null;
                                pointerSelectionRef.current = currency.code;
                                if (!pointerMovedRef.current) selectOption(currency.code);
                            }}
                            onClick={() => {
                                if (pointerSelectionRef.current !== null) {
                                    pointerSelectionRef.current = null;
                                    return;
                                }
                                selectOption(currency.code);
                            }}
                        >
                            <span className="min-w-0 flex-1 truncate font-medium">
                                {currency.code} — {currency.displayName}
                            </span>
                            {currency.code === value && <Icon path={mdiCheck} size={0.8} aria-label="Selected" />}
                            </button>
                        </Fragment>
                    ))}
                </div>
            )}
        </div>
    );
}
