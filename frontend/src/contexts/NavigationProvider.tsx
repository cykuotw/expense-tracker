import { useLayoutEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useLocation, useNavigationType } from "react-router-dom";
import { NavigationContext } from "../hooks/navigation";
import { NavigationStore } from "../lib/navigation";

export default function NavigationProvider({ children }: { children: ReactNode }) {
    const location = useLocation();
    const kind = useNavigationType();
    const [store] = useState(() => {
        const value = new NavigationStore();
        value.observe(location.key, `${location.pathname}${location.search}${location.hash}`, "POP");
        return value;
    });
    const [revision, setRevision] = useState(0);
    const context = useMemo(() => ({ store, revision }), [store, revision]);

    useLayoutEffect(() => {
        if (store.observe(location.key, `${location.pathname}${location.search}${location.hash}`, kind)) setRevision((value) => value + 1);
    }, [store, location, kind]);

    useLayoutEffect(() => {
        const capture = () => {
            const entry = store.entries.get(store.history[store.index]);
            if (!entry) return;
            const active = document.activeElement;
            entry.position = {
                top: window.scrollY, left: window.scrollX,
                href: active instanceof HTMLAnchorElement ? active.getAttribute("href") ?? undefined : undefined,
                id: active instanceof HTMLElement ? active.id || undefined : undefined,
            };
        };
        const captureLink = (event: MouseEvent) => {
            const link = event.target instanceof Element ? event.target.closest<HTMLAnchorElement>("a[href]") : null;
            if (!link) return;
            const entry = store.entries.get(store.history[store.index]);
            if (entry) entry.position = { top: window.scrollY, left: window.scrollX, href: link.getAttribute("href") ?? undefined };
        };
        window.addEventListener("scroll", capture, { passive: true });
        document.addEventListener("focusin", capture);
        document.addEventListener("click", captureLink, true);
        const previousRestoration = window.history.scrollRestoration;
        window.history.scrollRestoration = "manual";
        return () => {
            window.removeEventListener("scroll", capture);
            document.removeEventListener("focusin", capture);
            document.removeEventListener("click", captureLink, true);
            window.history.scrollRestoration = previousRestoration;
        };
    }, [store]);

    useLayoutEffect(() => {
        const entry = store.entries.get(location.key);
        const saved = entry?.position;
        let frame = 0;
        let timer = 0;
        let cancelled = false;
        const deadline = performance.now() + 10000;
        const restore = () => {
            if (cancelled) return;
            const blocked = (store.blockers.get(location.key)?.size ?? 0) > 0;
            const awaitingShell = !document.querySelector(".page-shell, .receipt-workspace-page") && performance.now() < deadline;
            if (blocked || awaitingShell) { timer = window.setTimeout(() => { frame = requestAnimationFrame(restore); }, 50); return; }
            const anchors = saved?.href ? Array.from(document.querySelectorAll<HTMLAnchorElement>("a[href]")).filter((anchor) => anchor.getAttribute("href") === saved.href) : [];
            const focus = (saved?.id ? document.getElementById(saved.id) : undefined) ?? anchors.find((anchor) => anchor.getClientRects().length > 0);
            if (focus) focus.focus({ preventScroll: true });
            else if (saved) {
                const heading = document.querySelector<HTMLElement>("#main-content h1");
                if (heading) { heading.tabIndex = -1; heading.focus({ preventScroll: true }); }
            }
            window.scrollTo({ top: saved?.top ?? 0, left: saved?.left ?? 0, behavior: "instant" });
        };
        frame = requestAnimationFrame(restore);
        return () => { cancelled = true; cancelAnimationFrame(frame); clearTimeout(timer); };
    }, [store, location.key, revision]);

    return <NavigationContext.Provider value={context}>{children}</NavigationContext.Provider>;
}
