import { useCallback, useEffect, useState } from "react";
import { apiFetch, asArray } from "../lib/api";
import { CurrencyMetadata } from "../lib/money";

export function useCurrencies() {
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    const load = useCallback(async (signal?: AbortSignal) => {
        setLoading(true);
        setError(false);
        try {
            const response = await apiFetch("/currencies", { signal });
            if (!response.ok) throw new Error("currency metadata request failed");
            const data = await response.json();
            if (signal?.aborted) return;
            setCurrencies(asArray<CurrencyMetadata>(data));
        } catch {
            if (signal?.aborted) return;
            setCurrencies([]);
            setError(true);
        } finally {
            if (!signal?.aborted) {
                setLoading(false);
            }
        }
    }, []);

    useEffect(() => {
        const abortController = new AbortController();
        void load(abortController.signal);
        return () => abortController.abort();
    }, [load]);

    return { currencies, loading, error, reload: load };
}
