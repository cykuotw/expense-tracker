import { useEffect, useState } from "react";
import { apiFetch, asArray } from "../lib/api";
import { CurrencyMetadata } from "../lib/money";

export function useCurrencies() {
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    const load = async () => {
        setLoading(true);
        setError(false);
        try {
            const response = await apiFetch("/currencies");
            if (!response.ok) throw new Error("currency metadata request failed");
            setCurrencies(asArray<CurrencyMetadata>(await response.json()));
        } catch {
            setCurrencies([]);
            setError(true);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        void load();
    }, []);

    return { currencies, loading, error, reload: load };
}
