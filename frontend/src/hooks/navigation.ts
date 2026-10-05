import { createContext, useCallback, useContext, useLayoutEffect, useState } from "react";
import type { Dispatch, MouseEvent, SetStateAction } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { destinationLabel, NavigationStore, routeFallback } from "../lib/navigation";
import type { ViewValue } from "../lib/navigation";

export const NavigationContext = createContext<{ store: NavigationStore; revision: number } | null>(null);

export function useReturnNavigation(fallback: string, fallbackLabel?: string) {
    const context = useContext(NavigationContext);
    const location = useLocation();
    const navigate = useNavigate();
    const url = `${location.pathname}${location.search}${location.hash}`;
    const target = context?.store.target(location.key, url, fallback);
    const to = target?.url ?? routeFallback(url, fallback);
    const label = fallbackLabel && (to === fallback || fallbackLabel.startsWith("Cancel")) ? fallbackLabel : destinationLabel(to);
    const finish = (destination?: string) => {
        const action = context?.store.returnAction(location.key, url, fallback, destination);
        if (action?.delta !== undefined) navigate(action.delta);
        else navigate(action?.url ?? routeFallback(url, destination ?? fallback), { replace: true });
    };
    const onClick = (event: MouseEvent<HTMLAnchorElement>) => {
        if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
        event.preventDefault();
        finish();
    };
    return { to, label, onClick, finish };
}

export function useNavigationState<T extends ViewValue>(name: string, initial: T): [T, Dispatch<SetStateAction<T>>] {
    const context = useContext(NavigationContext);
    const { key } = useLocation();
    const store = context?.store;
    const entry = store?.entries.get(key);
    const [local, setLocal] = useState<{ key: string; name: string; hydrated: boolean; value: T }>(() => ({ key, name, hydrated: !!entry, value: (entry?.view.get(name) as T | undefined) ?? initial }));
    const value = local.key === key && local.name === name && (local.hydrated || !entry) ? local.value : (entry?.view.get(name) as T | undefined) ?? initial;
    const setValue: Dispatch<SetStateAction<T>> = useCallback((next) => {
        setLocal((previous) => {
            const entry = store?.entries.get(key);
            const current = previous.key === key && previous.name === name && (previous.hydrated || !entry) ? previous.value : (entry?.view.get(name) as T | undefined) ?? initial;
            const result = typeof next === "function" ? next(current) : next;
            entry?.view.set(name, result);
            if (entry && entry.view.size > 200) entry.view.delete(entry.view.keys().next().value!);
            return { key, name, hydrated: !!entry, value: result };
        });
    }, [store, key, name, initial]);
    return [value, setValue];
}

export function useNavigationReady(ready: boolean) {
    const context = useContext(NavigationContext);
    const { key } = useLocation();
    useLayoutEffect(() => {
        if (!context || ready) return;
        const token = Symbol();
        const blockers = context.store.blockers.get(key) ?? new Set<symbol>();
        blockers.add(token);
        context.store.blockers.set(key, blockers);
        return () => { blockers.delete(token); };
    }, [context, key, ready]);
}
