import { useCallback, useRef, useState } from "react";
import { mdiCheck, mdiChevronDown } from "@mdi/js";
import Icon from "@mdi/react";
import { getGroupTypePresentation, groupTypeOptions } from "../../lib/groupTypePresentation";

interface GroupTypePickerProps {
    value: string;
    onChange: (value: string) => void;
}

export function GroupTypePicker({ value, onChange }: GroupTypePickerProps) {
    const [open, setOpen] = useState(false);
    const pointerSelectionRef = useRef<string | null>(null);
    const selected = getGroupTypePresentation(value);
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
                aria-label="Group type"
                aria-expanded={open}
                aria-haspopup="listbox"
                className="flex min-h-20 w-full items-center gap-3 rounded-2xl border border-border bg-background px-4 py-3 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                onClick={() => {
                    pointerSelectionRef.current = null;
                    setOpen((current) => !current);
                }}
            >
                <span className={`flex size-10 items-center justify-center rounded-xl ${selected.iconClassName}`}>
                    <Icon path={selected.icon} size={1} aria-hidden="true" />
                </span>
                <span className="min-w-0 flex-1">
                    <span className="block truncate font-semibold">{selected.label}</span>
                </span>
                <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
            </button>
            {open && (
                <div
                    role="listbox"
                    aria-label="Group type options"
                    className="absolute z-20 mt-2 w-full rounded-2xl border border-border bg-background p-2 shadow-xl"
                >
                    {groupTypeOptions.map((option) => (
                        <button
                            key={option.value}
                            type="button"
                            role="option"
                            aria-selected={option.value === value}
                            className={`flex min-h-11 w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${option.value === value ? "bg-primary/10" : ""}`}
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
                            <span className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${option.iconClassName}`}>
                                <Icon path={option.icon} size={0.8} aria-hidden="true" />
                            </span>
                            <span className="min-w-0 flex-1 truncate font-medium">{option.label}</span>
                            {option.value === value && <Icon path={mdiCheck} size={0.8} aria-label="Selected" />}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}
