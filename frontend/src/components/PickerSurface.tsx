import { cloneElement, useId, useLayoutEffect, useRef, useState } from "react";
import type { ComponentProps, KeyboardEvent, MouseEvent, ReactElement, ReactNode } from "react";
import { createPortal } from "react-dom";
import { getFormViewport, pickerPlacement, revealPickerOption } from "../lib/formViewport";

interface PickerSurfaceProps {
    open: boolean;
    onClose: () => void;
    trigger: ReactElement<ComponentProps<"button">>;
    label: string;
    children: ReactNode;
    search?: ReactNode;
    maxHeight?: number;
    preferAbove?: boolean;
}

export default function PickerSurface({ open, onClose, trigger, label, children, search, maxHeight = 320, preferAbove = false }: PickerSurfaceProps) {
    const triggerRef = useRef<HTMLButtonElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const listRef = useRef<HTMLDivElement>(null);
    const closeRef = useRef(onClose);
    closeRef.current = onClose;
    const id = useId();
    const revealedSelectionRef = useRef(false);
    const pendingFocusRef = useRef<HTMLElement | null>(null);
    const [position, setPosition] = useState<ReturnType<typeof pickerPlacement> | null>(null);

    useLayoutEffect(() => {
        if (!open) { setPosition(null); return; }
        const mountedPanel = panelRef.current;
        const mountedTrigger = triggerRef.current;
        let frame = 0;
        const update = () => {
            const anchor = triggerRef.current;
            const panel = panelRef.current;
            if (!anchor || !panel) return;
            const bounds = getFormViewport();
            const rect = anchor.getBoundingClientRect();
            if (rect.height > 0 && (rect.bottom < bounds.top || rect.top > bounds.bottom) &&
                !panel.contains(document.activeElement)) {
                closeRef.current();
                return;
            }
            const contentHeight = (listRef.current?.scrollHeight ?? 0) +
                (panel.querySelector<HTMLElement>("[data-picker-search]")?.offsetHeight ?? 0) + 24;
            const next = pickerPlacement(rect, bounds, Math.min(maxHeight, contentHeight || maxHeight), preferAbove);
            setPosition((current) => current && Object.entries(next).every(([key, value]) =>
                current[key as keyof typeof current] === value) ? current : next);
        };
        const schedule = () => {
            cancelAnimationFrame(frame);
            frame = requestAnimationFrame(update);
        };
        const outside = (event: PointerEvent) => {
            if (event.target instanceof Node && !triggerRef.current?.contains(event.target) &&
                !panelRef.current?.contains(event.target)) closeRef.current();
        };
        update();
        document.addEventListener("pointerdown", outside);
        window.addEventListener("scroll", schedule, true);
        window.addEventListener("resize", schedule);
        window.visualViewport?.addEventListener("resize", schedule);
        window.visualViewport?.addEventListener("scroll", schedule);
        const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(schedule);
        if (triggerRef.current) observer?.observe(triggerRef.current);
        if (listRef.current) observer?.observe(listRef.current);
        return () => {
            cancelAnimationFrame(frame);
            observer?.disconnect();
            if (mountedPanel?.contains(document.activeElement)) mountedTrigger?.focus({ preventScroll: true });
            document.removeEventListener("pointerdown", outside);
            window.removeEventListener("scroll", schedule, true);
            window.removeEventListener("resize", schedule);
            window.visualViewport?.removeEventListener("resize", schedule);
            window.visualViewport?.removeEventListener("scroll", schedule);
        };
    }, [open, maxHeight, preferAbove]);

    useLayoutEffect(() => {
        if (!open) { revealedSelectionRef.current = false; return; }
        if (!position || !listRef.current || revealedSelectionRef.current) return;
        revealedSelectionRef.current = true;
        const selected = listRef.current.querySelector<HTMLElement>('[aria-selected="true"]');
        if (selected) revealPickerOption(selected, listRef.current, false);
        // Reveal once on open, without resetting the user's position after filtering/resizing.
    });

    useLayoutEffect(() => {
        if (!open) { pendingFocusRef.current = null; return; }
        if (position && pendingFocusRef.current) {
            pendingFocusRef.current.focus({ preventScroll: true });
            pendingFocusRef.current = null;
        }
    }, [open, position]);

    const options = () => Array.from(listRef.current?.querySelectorAll<HTMLButtonElement>('[role="option"]:not(:disabled)') ?? []);
    const focusOption = (option?: HTMLElement) => {
        if (!option) return;
        // A fast key press may arrive before the portal's measured position is committed.
        if (getComputedStyle(option).visibility === "hidden") pendingFocusRef.current = option;
        else option.focus({ preventScroll: true });
    };
    const keyboard = (event: KeyboardEvent<HTMLDivElement>) => {
        if (!open) return;
        if (event.key === "Escape") {
            event.preventDefault();
            closeRef.current();
            triggerRef.current?.focus({ preventScroll: true });
            return;
        }
        const target = event.target as HTMLElement;
        const isTextInput = target.matches("input, textarea");
        const items = options();
        if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key) && (!isTextInput || event.key === "ArrowDown")) {
            event.preventDefault();
            const index = items.indexOf(target as HTMLButtonElement);
            const selectedIndex = items.findIndex((item) => item.getAttribute("aria-selected") === "true");
            const nextIndex = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 :
                index < 0 ? (selectedIndex < 0 ? (event.key === "ArrowUp" ? items.length - 1 : 0) : selectedIndex) :
                Math.max(0, Math.min(items.length - 1, index + (event.key === "ArrowDown" ? 1 : -1)));
            focusOption(items[nextIndex]);
        }
        if (event.key === "Tab") {
            const searchInput = panelRef.current?.querySelector<HTMLInputElement>("input");
            if (target === triggerRef.current && !event.shiftKey) {
                if (!searchInput && items.length === 0) { closeRef.current(); return; }
                event.preventDefault();
                if (searchInput) searchInput.focus({ preventScroll: true });
                else focusOption(items.find((item) => item.getAttribute("aria-selected") === "true") ?? items[0]);
            } else if (panelRef.current?.contains(target)) {
                event.preventDefault();
                if (target === searchInput && !event.shiftKey && items.length > 0) {
                    focusOption(items.find((item) => item.getAttribute("aria-selected") === "true") ?? items[0]);
                } else if (event.shiftKey && target !== searchInput && searchInput) {
                    searchInput.focus({ preventScroll: true });
                } else if (event.shiftKey) {
                    triggerRef.current?.focus({ preventScroll: true });
                } else {
                    const tabbables = Array.from(document.querySelectorAll<HTMLElement>(
                        'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]',
                    )).filter((item) => !panelRef.current?.contains(item) && item.tabIndex >= 0 && item.getClientRects().length > 0);
                    const index = tabbables.indexOf(triggerRef.current!);
                    closeRef.current();
                    tabbables[index + 1]?.focus();
                }
            }
        }
    };

    return (
        <div className="relative" onKeyDown={keyboard} onBlur={(event) => {
            const next = event.relatedTarget;
            if (next instanceof Node && (triggerRef.current?.contains(next) || panelRef.current?.contains(next))) return;
            // Touch may temporarily clear focus while scrolling the portal.
            if (next !== null) closeRef.current();
        }}>
            {cloneElement(trigger, {
                ref: triggerRef,
                "aria-controls": open ? id : undefined,
                onClick: (event: MouseEvent<HTMLButtonElement>) => {
                    if (!open && triggerRef.current) {
                        setPosition(pickerPlacement(triggerRef.current.getBoundingClientRect(), getFormViewport(), maxHeight, preferAbove));
                    }
                    trigger.props.onClick?.(event);
                },
            } as ComponentProps<"button">)}
            {open && createPortal(
                <div ref={panelRef} data-picker-panel className="form-picker-panel" style={{
                    top: position?.top ?? 0,
                    left: position?.left ?? 0,
                    width: position?.width ?? 0,
                    maxHeight: position?.height ?? maxHeight,
                    visibility: position ? "visible" : "hidden",
                }} data-side={position?.side}>
                    {search ? <div data-picker-search className="shrink-0 pb-3">{search}</div> : null}
                    <div ref={listRef} id={id} role="listbox" aria-label={label} className="min-h-0 overflow-y-auto overscroll-contain touch-pan-y" onFocus={(event) => {
                        if (event.target.matches('[role="option"]') && listRef.current) revealPickerOption(event.target, listRef.current);
                    }}>{children}</div>
                </div>, document.body,
            )}
        </div>
    );
}
