import { useSyncExternalStore } from "react";

let unavailable = false;
let recoveryNeeded = false;
const listeners = new Set<() => void>();

export function setServiceUnavailable(value: boolean) {
    if (value === unavailable) return;
    unavailable = value;
    if (value) recoveryNeeded = true;
    listeners.forEach((notify) => notify());
}

function subscribe(notify: () => void) {
    listeners.add(notify);
    return () => { listeners.delete(notify); };
}

export function useServiceUnavailable() {
    return useSyncExternalStore(subscribe, () => unavailable);
}

export function useServiceRecoveryNeeded() {
    return useSyncExternalStore(subscribe, () => recoveryNeeded);
}

export function dismissServiceRecovery() {
    recoveryNeeded = false;
    listeners.forEach((notify) => notify());
}

export class ServiceUnavailableError extends Error {
    constructor() {
        super("Service unavailable. Your request may not have completed. Check the result before submitting again.");
        this.name = "ServiceUnavailableError";
    }
}

export async function serviceFetch(input: string, init: RequestInit = {}) {
    const controller = new AbortController();
    const cancel = () => controller.abort(init.signal?.reason);
    if (init.signal?.aborted) cancel();
    else init.signal?.addEventListener("abort", cancel, { once: true });
    const timeout = setTimeout(() => controller.abort(new DOMException("Service request timed out", "TimeoutError")), 15_000);
    try {
        const response = await fetch(input, {
            ...init,
            signal: controller.signal,
        });
        if (response.status >= 500) {
            setServiceUnavailable(true);
            throw new ServiceUnavailableError();
        }
        if (response.ok) setServiceUnavailable(false);
        return response;
    } catch (error) {
        if (!init.signal?.aborted) {
            setServiceUnavailable(true);
        }
        throw error;
    } finally {
        clearTimeout(timeout);
        init.signal?.removeEventListener("abort", cancel);
    }
}
