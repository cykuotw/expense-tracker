import { useCallback, useEffect, useRef, useState } from "react";
import { mdiCheck, mdiChevronDown } from "@mdi/js";
import Icon from "@mdi/react";
import {
    getGroupTypePresentation,
    groupTypeOptions,
} from "../../lib/groupTypePresentation";

interface GroupTypePickerProps {
    value: string;
    onChange: (value: string) => void;
}

const mobileMediaQuery = "(max-width: 767px)";

function useMobileViewport() {
    const [isMobile, setIsMobile] = useState(
        () => typeof window !== "undefined" && window.matchMedia?.(mobileMediaQuery).matches
    );

    useEffect(() => {
        const media = window.matchMedia?.(mobileMediaQuery);
        if (!media) return;

        const update = () => setIsMobile(media.matches);
        update();
        if (media.addEventListener) {
            media.addEventListener("change", update);
            return () => media.removeEventListener("change", update);
        }
        media.addListener(update);
        return () => media.removeListener(update);
    }, []);

    return isMobile;
}

export function GroupTypePicker({ value, onChange }: GroupTypePickerProps) {
    const isMobile = useMobileViewport();
    const [open, setOpen] = useState(false);
    const [mobileOpen, setMobileOpen] = useState(false);
    const [draftValue, setDraftValue] = useState(value);
    const pointerSelectionRef = useRef<string | null>(null);
    const selected = getGroupTypePresentation(value);
    const draft = getGroupTypePresentation(draftValue);
    const selectOption = useCallback(
        (nextValue: string) => {
            onChange(nextValue);
            setOpen(false);
        },
        [onChange]
    );
    const closeMobilePicker = useCallback(() => {
        setDraftValue(value);
        setMobileOpen(false);
    }, [value]);

    useEffect(() => {
        if (!mobileOpen) return;
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === "Escape") closeMobilePicker();
        };
        document.addEventListener("keydown", closeOnEscape);
        return () => document.removeEventListener("keydown", closeOnEscape);
    }, [closeMobilePicker, mobileOpen]);

    if (isMobile) {
        return (
            <>
                <button
                    type="button"
                    aria-label="Group type"
                    aria-expanded={mobileOpen}
                    aria-haspopup="dialog"
                    className="flex min-h-14 w-full items-center gap-3 rounded-2xl border border-border bg-background px-4 py-2 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                    onClick={() => {
                        setDraftValue(value);
                        setMobileOpen(true);
                    }}
                >
                    <span className={`flex size-9 items-center justify-center rounded-xl ${selected.iconClassName}`}>
                        <Icon path={selected.icon} size={0.9} aria-hidden="true" />
                    </span>
                    <span className="min-w-0 flex-1">
                        <span className="block truncate font-semibold">{selected.label}</span>
                    </span>
                    <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
                </button>
                {mobileOpen && (
                    <div
                        className="fixed inset-0 z-50 flex items-end bg-foreground/45 p-3 sm:items-center sm:justify-center"
                        onPointerDown={(event) => {
                            if (event.target === event.currentTarget) closeMobilePicker();
                        }}
                    >
                        <section
                            role="dialog"
                            aria-modal="true"
                            aria-labelledby="group-type-picker-title"
                            className="max-h-[min(42rem,calc(100dvh-1.5rem))] w-full max-w-lg overflow-y-auto rounded-[1.75rem] border border-border bg-background p-4 shadow-2xl"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div>
                                    <h2 id="group-type-picker-title" className="text-lg font-bold">Choose group type</h2>
                                    <p className="mt-1 text-sm text-foreground/60">Choose a type, then confirm it.</p>
                                </div>
                                <span className={`flex size-10 shrink-0 items-center justify-center rounded-xl ${draft.iconClassName}`}>
                                    <Icon path={draft.icon} size={0.95} aria-hidden="true" />
                                </span>
                            </div>
                            <div role="radiogroup" aria-label="Group type options" className="mt-4 grid gap-2">
                                {groupTypeOptions.map((option) => (
                                    <button
                                        key={option.value}
                                        type="button"
                                        role="radio"
                                        aria-checked={option.value === draftValue}
                                        className={`flex min-h-12 w-full items-center gap-3 rounded-xl border px-3 py-2 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${option.value === draftValue ? "border-primary bg-primary/10" : "border-transparent bg-muted/65 hover:bg-muted"}`}
                                        onClick={() => setDraftValue(option.value)}
                                    >
                                        <span className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${option.iconClassName}`}>
                                            <Icon path={option.icon} size={0.8} aria-hidden="true" />
                                        </span>
                                        <span className="min-w-0 flex-1 truncate font-medium">{option.label}</span>
                                        {option.value === draftValue && <Icon path={mdiCheck} size={0.8} aria-label="Selected" />}
                                    </button>
                                ))}
                            </div>
                            <div className="mt-4 grid grid-cols-2 gap-2 border-t border-border pt-4">
                                <button type="button" className="ui-button ui-button-ghost min-h-12" onClick={closeMobilePicker}>
                                    Cancel
                                </button>
                                <button
                                    type="button"
                                    className="ui-button ui-button-primary min-h-12"
                                    onClick={() => {
                                        onChange(draftValue);
                                        setMobileOpen(false);
                                    }}
                                >
                                    Use {draft.label}
                                </button>
                            </div>
                        </section>
                    </div>
                )}
            </>
        );
    }

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
