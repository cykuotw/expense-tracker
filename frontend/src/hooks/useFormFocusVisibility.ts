import { useEffect, useRef } from "react";
import { formScrollBehavior, getFormViewport, revealDelta } from "../lib/formViewport";

const controls = "input:not([type=hidden]), textarea, select, button, a[href], [tabindex]";

export function useFormFocusVisibility() {
    const pageRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const page = pageRef.current;
        if (!page) return;
        let timer: ReturnType<typeof setTimeout> | undefined;
        let manuallyScrolled = false;
        let invalidField: HTMLElement | null = null;

        const reveal = () => {
            const target = document.activeElement;
            if (!(target instanceof HTMLElement) || !page.contains(target) ||
                target.closest("[data-picker-panel], dialog") || manuallyScrolled) return;

            const rect = target.getBoundingClientRect();
            if (!rect.height) return;
            const shellRect = target.closest(".expense-form-input-shell, .ui-input-shell, .expense-form-date-input")?.getBoundingClientRect();
            const controlRect = shellRect?.height ? shellRect : rect;
            const field = target.closest('[data-slot="field"], [data-form-field]') ??
                target.closest(".expense-form-date-input")?.parentElement ??
                target.closest("label")?.parentElement;
            let top = controlRect.top;
            let bottom = controlRect.bottom;
            if (field) {
                for (const label of field.querySelectorAll("label")) {
                    const labelRect = label.getBoundingClientRect();
                    if (labelRect.height) top = Math.min(top, labelRect.top);
                }
            }
            for (const id of (target.getAttribute("aria-describedby") ?? "").split(/\s+/)) {
                const feedback = id ? document.getElementById(id) : null;
                if (feedback && page.contains(feedback)) {
                    const feedbackRect = feedback.getBoundingClientRect();
                    if (feedbackRect.height) bottom = Math.max(bottom, feedbackRect.bottom);
                }
            }
            const bounds = getFormViewport();
            // If a large label/feedback group cannot fit, keep the actual input usable.
            const delta = revealDelta(bottom - top > bounds.bottom - bounds.top ? controlRect : { top, bottom }, bounds);
            if (Math.abs(delta) > 1) window.scrollBy({ top: delta, behavior: formScrollBehavior() });
        };
        const schedule = () => {
            clearTimeout(timer);
            // Let native focus scrolling and software-keyboard resizing settle first.
            timer = setTimeout(reveal, 150);
        };
        const focus = (event: FocusEvent) => {
            if (!(event.target instanceof HTMLElement) || !event.target.matches(controls)) return;
            manuallyScrolled = false;
            schedule();
        };
        const manualScroll = () => {
            manuallyScrolled = true;
            clearTimeout(timer);
        };
        const resize = () => {
            const viewport = window.visualViewport;
            const keyboardSpace = viewport && viewport.scale === 1
                ? Math.max(0, window.innerHeight - viewport.height - viewport.offsetTop)
                : 0;
            page.style.setProperty("--form-keyboard-space", `${keyboardSpace}px`);
            if (!manuallyScrolled) schedule();
        };
        const invalid = (event: Event) => {
            if (invalidField || !(event.target instanceof HTMLElement)) return;
            invalidField = event.target;
            clearTimeout(timer);
            timer = setTimeout(() => {
                invalidField?.focus({ preventScroll: true });
                invalidField = null;
                manuallyScrolled = false;
                schedule();
            }, 0);
        };

        page.addEventListener("focusin", focus);
        page.addEventListener("invalid", invalid, true);
        window.addEventListener("resize", resize);
        window.visualViewport?.addEventListener("resize", resize);
        window.addEventListener("wheel", manualScroll, { passive: true });
        window.addEventListener("touchmove", manualScroll, { passive: true });
        resize();
        return () => {
            clearTimeout(timer);
            page.removeEventListener("focusin", focus);
            page.removeEventListener("invalid", invalid, true);
            window.removeEventListener("resize", resize);
            window.visualViewport?.removeEventListener("resize", resize);
            window.removeEventListener("wheel", manualScroll);
            window.removeEventListener("touchmove", manualScroll);
            page.style.removeProperty("--form-keyboard-space");
        };
    }, []);

    return pageRef;
}
