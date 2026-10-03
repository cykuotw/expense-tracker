import PickerSurface from "../PickerSurface";
import { useCallback, useRef, useState } from "react";
import { mdiCheck, mdiChevronDown } from "@mdi/js";
import Icon from "@mdi/react";
import {
    getGroupTypePresentation,
    groupTypeOptions,
} from "../../lib/groupTypePresentation";

interface GroupTypePickerProps {
    value: string;
    onChange: (value: string) => void;
    disabled?: boolean;
}

export function GroupTypePicker({ value, onChange, disabled = false }: GroupTypePickerProps) {
    const [open, setOpen] = useState(false);
    const pointerSelectionRef = useRef<string | null>(null);
    const pointerStartYRef = useRef<number | null>(null);
    const pointerMovedRef = useRef(false);
    const selected = getGroupTypePresentation(value);
    const selectOption = useCallback(
        (nextValue: string) => {
            onChange(nextValue);
            setOpen(false);
        },
        [onChange]
    );

    return (
        <PickerSurface open={open} onClose={() => setOpen(false)} label="Group type options"
            trigger={
                <button
                    type="button"
                    aria-label="Group type"
                    aria-expanded={open}
                    aria-haspopup="listbox"
                    disabled={disabled}
                    className="flex min-h-14 w-full touch-manipulation items-center gap-3 rounded-2xl border border-border bg-background px-4 py-2 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:cursor-not-allowed disabled:opacity-60 md:min-h-20 md:py-3"
                    onClick={() => {
                        pointerSelectionRef.current = null;
                        setOpen((current) => !current);
                    }}
                >
                    <span className={`flex size-9 items-center justify-center rounded-xl md:size-10 ${selected.iconClassName}`}>
                        <Icon path={selected.icon} size={0.9} aria-hidden="true" />
                    </span>
                    <span className="min-w-0 flex-1">
                        <span className="block truncate font-semibold">{selected.label}</span>
                    </span>
                    <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
                </button>
            }
        >
            {groupTypeOptions.map((option) => (
                <button
                    key={option.value}
                    type="button"
                    tabIndex={-1}
                    role="option"
                    aria-selected={option.value === value}
                    className={`flex min-h-12 w-full touch-manipulation items-center gap-3 rounded-xl px-3 py-2.5 text-left hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${option.value === value ? "bg-primary/10" : ""}`}
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
                        pointerSelectionRef.current = option.value;
                        if (pointerMovedRef.current) return;
                        selectOption(option.value);
                    }}
                    onClick={() => {
                        if (pointerSelectionRef.current !== null) {
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
        </PickerSurface>
    );
}
