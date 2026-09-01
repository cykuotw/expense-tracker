import { useCallback, useRef, useState } from "react";
import { mdiCheck, mdiChevronDown } from "@mdi/js";
import Icon from "@mdi/react";

export interface ExpenseFormPickerOption {
    value: string;
    label: string;
    description?: string;
    icon?: string;
}

interface ExpenseFormPickerProps {
    label: string;
    options: ExpenseFormPickerOption[];
    value: string;
    onChange: (value: string) => void;
    icon?: string;
    emptyLabel: string;
}

export function ExpenseFormPicker({
    label,
    options,
    value,
    onChange,
    icon,
    emptyLabel,
}: ExpenseFormPickerProps) {
    const [open, setOpen] = useState(false);
    const pointerSelectionRef = useRef<string | null>(null);
    const selected = options.find((option) => option.value === value);
    const selectedIcon = selected?.icon ?? icon;
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
                if (!event.currentTarget.contains(event.relatedTarget)) {
                    setOpen(false);
                }
            }}
            onKeyDown={(event) => {
                if (event.key === "Escape") setOpen(false);
            }}
        >
            <button
                type="button"
                aria-label={label}
                aria-expanded={open}
                aria-haspopup="listbox"
                className="flex min-h-14 w-full items-center gap-3 rounded-2xl border border-border bg-background px-4 py-2 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary md:min-h-20 md:py-3"
                onClick={() => {
                    pointerSelectionRef.current = null;
                    setOpen((current) => !current);
                }}
            >
                {selectedIcon && (
                    <span className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                        <Icon path={selectedIcon} size={1} />
                    </span>
                )}
                <span className="min-w-0 flex-1">
                    <span className="block truncate font-semibold">
                        {selected?.label ?? emptyLabel}
                    </span>
                    {selected?.description && (
                        <span className="block truncate text-xs text-foreground/60">
                            {selected.description}
                        </span>
                    )}
                </span>
                <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
            </button>
            {open && (
                <div
                    role="listbox"
                    aria-label={`${label} options`}
                    className="absolute z-20 mt-2 max-h-80 w-full overflow-y-auto rounded-2xl border border-border bg-background p-2 shadow-xl"
                >
                    {options.length === 0 ? (
                        <p className="px-3 py-2 text-sm text-foreground/60">
                            No options available.
                        </p>
                    ) : (
                        options.map((option) => (
                            <button
                                key={option.value}
                                type="button"
                                role="option"
                                aria-selected={option.value === value}
                                className={`flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${option.value === value ? "bg-primary/10" : ""}`}
                                onPointerDown={(event) => {
                                    event.preventDefault();
                                    pointerSelectionRef.current = option.value;
                                    selectOption(option.value);
                                }}
                                onClick={() => {
                                    if (pointerSelectionRef.current === option.value) {
                                        pointerSelectionRef.current = null;
                                        return;
                                    }
                                    selectOption(option.value);
                                }}
                            >
                                {option.icon ?? icon ? (
                                    <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                                        <Icon
                                            path={option.icon ?? icon ?? ""}
                                            size={0.8}
                                        />
                                    </span>
                                ) : null}
                                <span className="min-w-0 flex-1">
                                    <span className="block truncate font-medium">
                                        {option.label}
                                    </span>
                                    {option.description && (
                                        <span className="block truncate text-xs text-foreground/60">
                                            {option.description}
                                        </span>
                                    )}
                                </span>
                                {option.value === value && (
                                    <Icon
                                        path={mdiCheck}
                                        size={0.8}
                                        aria-label="Selected"
                                    />
                                )}
                            </button>
                        ))
                    )}
                </div>
            )}
        </div>
    );
}
