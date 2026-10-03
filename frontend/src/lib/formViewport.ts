export interface VisibleBounds {
    top: number;
    bottom: number;
    left: number;
    right: number;
}

const margin = 12;

export function getFormViewport(): VisibleBounds {
    const viewport = window.visualViewport;
    const top = viewport?.offsetTop ?? 0;
    const left = viewport?.offsetLeft ?? 0;
    const bounds = {
        top: top + margin,
        bottom: top + (viewport?.height ?? window.innerHeight) - margin,
        left: left + margin,
        right: left + (viewport?.width ?? window.innerWidth) - margin,
    };
    for (const element of document.querySelectorAll<HTMLElement>(
        ".app-shell__mobile-nav, .mobile-page-header--sticky",
    )) {
        const rect = element.getBoundingClientRect();
        if (rect.height === 0 || rect.bottom <= bounds.top || rect.top >= bounds.bottom) continue;
        if (element.matches(".app-shell__mobile-nav")) {
            bounds.bottom = Math.min(bounds.bottom, rect.top - margin);
        } else if (rect.top <= top + margin) {
            bounds.top = Math.max(bounds.top, rect.bottom + margin);
        }
    }
    return bounds;
}

export function revealDelta(rect: Pick<DOMRect, "top" | "bottom">, bounds: Pick<VisibleBounds, "top" | "bottom">): number {
    // An oversized field cannot fit; prioritize its beginning instead of oscillating.
    if (rect.bottom - rect.top > bounds.bottom - bounds.top) return rect.top - bounds.top;
    if (rect.top < bounds.top) return rect.top - bounds.top;
    if (rect.bottom > bounds.bottom) return rect.bottom - bounds.bottom;
    return 0;
}

export function pickerPlacement(
    anchor: Pick<DOMRect, "top" | "bottom" | "left" | "width">,
    bounds: VisibleBounds,
    desiredHeight: number,
    preferAbove = false,
) {
    const gap = 8;
    const above = Math.max(0, anchor.top - bounds.top - gap);
    const below = Math.max(0, bounds.bottom - anchor.bottom - gap);
    const useAbove = preferAbove
        ? above >= desiredHeight || above > below
        : below < desiredHeight && above > below;
    const viewportHeight = Math.max(0, bounds.bottom - bounds.top);
    let height = Math.min(desiredHeight, useAbove ? above : below, viewportHeight);
    // On short/keyboard viewports, use the visible area rather than squeezing the
    // search field and options into an unusably small strip next to the trigger.
    if (height < Math.min(160, desiredHeight)) height = Math.min(desiredHeight, viewportHeight);
    const width = Math.min(anchor.width, Math.max(0, bounds.right - bounds.left));
    return {
        side: useAbove ? "above" : "below",
        top: Math.max(bounds.top, Math.min(
            useAbove ? anchor.top - gap - height : anchor.bottom + gap,
            bounds.bottom - height,
        )),
        left: Math.max(bounds.left, Math.min(anchor.left, bounds.right - width)),
        width,
        height,
    };
}

export function formScrollBehavior(): ScrollBehavior {
    return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ? "instant" : "smooth";
}

export function revealPickerOption(option: HTMLElement, list: HTMLElement, animate = true) {
    const rect = list.getBoundingClientRect();
    if (rect.height === 0 || option.getBoundingClientRect().height === 0) return;
    const delta = revealDelta(option.getBoundingClientRect(), { top: rect.top + 4, bottom: rect.bottom - 4 });
    if (delta !== 0) list.scrollBy({ top: delta, behavior: animate ? formScrollBehavior() : "instant" });
}
