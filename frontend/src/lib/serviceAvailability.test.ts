import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { apiFetch, setApiAuthFailureHandler } from "./api";
import { ServiceUnavailableError, serviceFetch, setServiceUnavailable, useServiceUnavailable } from "./serviceAvailability";
import { act, cleanup, renderHook } from "@testing-library/react";

describe("service failures and authentication", () => {
    const expired = vi.fn();
    beforeEach(() => { setApiAuthFailureHandler(expired); expired.mockClear(); setServiceUnavailable(false); });
    afterEach(() => { cleanup(); setApiAuthFailureHandler(null); setServiceUnavailable(false); vi.unstubAllGlobals(); vi.useRealTimers(); });

    it("times out requests with caller signals and distinguishes intentional cancellation", async () => {
        vi.useFakeTimers();
        vi.stubGlobal("fetch", vi.fn((_input, init: RequestInit) => new Promise((_resolve, reject) => {
            init.signal?.addEventListener("abort", () => reject(init.signal?.reason), { once: true });
        })));
        const state = renderHook(useServiceUnavailable);
        const caller = new AbortController();
        const timedOut = expect(serviceFetch("/pending", { signal: caller.signal })).rejects.toMatchObject({ name: "TimeoutError" });
        await act(async () => { await vi.advanceTimersByTimeAsync(15_000); await timedOut; });
        expect(state.result.current).toBe(true);
        act(() => setServiceUnavailable(false));
        const cancelled = expect(serviceFetch("/cancelled", { signal: caller.signal })).rejects.toMatchObject({ name: "AbortError" });
        await act(async () => { caller.abort(); await cancelled; });
        expect(state.result.current).toBe(false);
    });

    it("preserves the session when refresh returns a service error", async () => {
        const fetch = vi.fn()
            .mockResolvedValueOnce(new Response(null, { status: 401 }))
            .mockResolvedValueOnce(new Response('{"csrfToken":"token"}', { status: 200 }))
            .mockResolvedValueOnce(new Response(null, { status: 503 }));
        vi.stubGlobal("fetch", fetch);
        await expect(apiFetch("/auth/me")).rejects.toBeInstanceOf(ServiceUnavailableError);
        expect(expired).not.toHaveBeenCalled();
        expect(fetch).toHaveBeenCalledTimes(3);
    });

    it("does not replay a mutation with an ambiguous service failure", async () => {
        const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 502 }));
        vi.stubGlobal("fetch", fetch);
        await expect(apiFetch("/expense", { method: "POST", body: "{}" })).rejects.toBeInstanceOf(ServiceUnavailableError);
        expect(expired).not.toHaveBeenCalled();
        expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("allows manual out-of-hours success and keeps application validation errors local", async () => {
        const fetch = vi.fn().mockResolvedValueOnce(new Response("{}", { status: 200 }))
            .mockResolvedValueOnce(new Response("{}", { status: 400 }));
        vi.stubGlobal("fetch", fetch);
        expect((await apiFetch("/groups")).status).toBe(200);
        expect((await apiFetch("/groups")).status).toBe(400);
        expect(expired).not.toHaveBeenCalled();
    });
});
