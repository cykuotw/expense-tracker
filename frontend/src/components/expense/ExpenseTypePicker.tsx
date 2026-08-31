import { useCallback, useMemo, useRef, useState } from "react";
import Icon from "@mdi/react";
import { mdiChevronDown, mdiMagnify } from "@mdi/js";
import { ExpenseTypeItem } from "../../types/expense";
import {
    getExpenseCategoryPresentation,
    getExpenseTypePresentation,
} from "../../lib/expenseCategoryPresentation";

export function ExpenseTypePicker({
    expenseTypes,
    value,
    onChange,
}: {
    expenseTypes: ExpenseTypeItem[];
    value: string;
    onChange: (id: string) => void;
}) {
    const [open, setOpen] = useState(false);
    const [query, setQuery] = useState("");
    const pointerSelectionRef = useRef<string | null>(null);
    const pointerStartYRef = useRef<number | null>(null);
    const pointerMovedRef = useRef(false);
    const selectType = useCallback(
        (id: string) => {
            onChange(id);
            setOpen(false);
            setQuery("");
        },
        [onChange]
    );
    const selected = expenseTypes.find((type) => type.id === value);
    const selectedPresentation = getExpenseTypePresentation(
        selected?.category,
        selected?.name
    );
    const groupedTypes = useMemo(() => {
        const normalizedQuery = query.trim().toLowerCase();
        return expenseTypes.reduce<Record<string, ExpenseTypeItem[]>>((groups, type) => {
            if (
                normalizedQuery &&
                !`${type.category} ${type.name}`.toLowerCase().includes(normalizedQuery)
            ) return groups;
            (groups[type.category] ??= []).push(type);
            return groups;
        }, {});
    }, [expenseTypes, query]);

    return (
        <div
            className="relative"
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) {
                    setOpen(false);
                    setQuery("");
                }
            }}
            onKeyDown={(event) => {
                if (event.key === "Escape") {
                    setOpen(false);
                    setQuery("");
                }
            }}
        >
            <button
                type="button"
                aria-expanded={open}
                aria-haspopup="listbox"
                className="flex min-h-20 w-full items-center gap-3 rounded-2xl border border-border bg-background px-4 py-3 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                onClick={() => {
                    pointerSelectionRef.current = null;
                    setOpen((current) => !current);
                }}
            >
                    <span
                        className={`flex size-10 items-center justify-center rounded-xl ${selectedPresentation.iconClassName}`}
                    >
                    <Icon path={selectedPresentation.icon} size={1} />
                </span>
                <span className="min-w-0 flex-1">
                    <span className="block truncate font-semibold">{selected?.name ?? "Choose a category"}</span>
                    <span className="block truncate text-xs text-foreground/60">{selected?.category ?? "Search or browse categories"}</span>
                </span>
                <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
            </button>
            {open && (
                <div
                    role="listbox"
                    aria-label="Expense type options"
                    className="absolute z-20 mt-2 max-h-96 w-full touch-pan-y overflow-y-auto overscroll-contain rounded-2xl border border-border bg-background p-3 shadow-xl"
                >
                    <label className="ui-input-shell flex items-center gap-2 bg-muted">
                        <Icon path={mdiMagnify} size={0.9} aria-hidden="true" />
                        <input
                            value={query}
                            onChange={(event) => setQuery(event.target.value)}
                            placeholder="Search categories"
                            className="grow"
                        />
                    </label>
                    <div className="mt-3 space-y-4">
                        {Object.entries(groupedTypes).map(([category, types]) => {
                            const presentation = getExpenseCategoryPresentation(category);
                            return (
                                <section key={category}>
                                    <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-foreground/60">
                                        {presentation.label}
                                    </h3>
                                    <div className="mt-2 grid gap-1">
                                        {types.map((type) => {
                                            const typePresentation = getExpenseTypePresentation(
                                                type.category,
                                                type.name
                                            );

                                            return (
                                                <button
                                                    key={type.id}
                                                    type="button"
                                                    role="option"
                                                    aria-selected={type.id === value}
                                                    className={`flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${type.id === value ? "bg-primary/10" : ""}`}
                                                    onPointerDown={(event) => {
                                                        pointerStartYRef.current = event.clientY;
                                                        pointerMovedRef.current = false;
                                                    }}
                                                    onPointerMove={(event) => {
                                                        if (
                                                            pointerStartYRef.current !== null &&
                                                            Math.abs(event.clientY - pointerStartYRef.current) > 8
                                                        ) {
                                                            pointerMovedRef.current = true;
                                                        }
                                                    }}
                                                    onPointerUp={() => {
                                                        pointerStartYRef.current = null;
                                                        pointerSelectionRef.current = type.id;
                                                        if (pointerMovedRef.current) return;
                                                        selectType(type.id);
                                                    }}
                                                    onClick={() => {
                                                        if (pointerSelectionRef.current !== null) {
                                                            pointerSelectionRef.current = null;
                                                            return;
                                                        }
                                                        selectType(type.id);
                                                    }}
                                                >
                                                    <span
                                                        className={`flex size-8 items-center justify-center rounded-lg ${typePresentation.iconClassName}`}
                                                    >
                                                        <Icon
                                                            path={typePresentation.icon}
                                                            size={0.8}
                                                        />
                                                    </span>
                                                    <span className="font-medium">
                                                        {type.name}
                                                    </span>
                                                </button>
                                            );
                                        })}
                                    </div>
                                </section>
                            );
                        })}
                    </div>
                </div>
            )}
        </div>
    );
}
