import { useEffect, useMemo, useRef, useState } from "react";
import { mdiRefresh, mdiTrashCanOutline } from "@mdi/js";
import Icon from "@mdi/react";

import { Input } from "@/components/ui/input";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Button } from "@/components/ui/button";
import { CurrencyMetadata } from "../../lib/money";
import {
    GroupCurrencySetting,
    GroupCurrencySettings,
} from "../../types/group";
import { CurrencyPicker } from "./CurrencyPicker";
import {
    compactRate,
    divideRates,
    fetchRateRecommendations,
    reciprocalRate,
    validPositiveRate,
    type Recommendation,
    type RecommendationResponse,
} from "../../lib/currencySettings";

type RateDirection = "source-to-preview" | "preview-to-source";

interface CurrencySettingsEditorProps {
    currencies: CurrencyMetadata[];
    settings: GroupCurrencySettings;
    onChange: (settings: GroupCurrencySettings) => void;
    disabled?: boolean;
}

function recommendationForCurrency(
    response: RecommendationResponse | null,
    sourceCurrency: string,
    targetCurrency: string
): Recommendation | null {
    const snapshotTarget = response?.recommendations[0]?.settlementPreviewCurrency;
    if (!response || !snapshotTarget) return null;
    const source = response.recommendations.find(
        (item) => item.sourceCurrency === sourceCurrency
    );
    const target = response.recommendations.find(
        (item) => item.sourceCurrency === targetCurrency
    );
    if (!source || !target) return null;

    const derivedRate = sourceCurrency === targetCurrency
        ? "1"
        : targetCurrency === snapshotTarget
            ? source.recommendedRate
            : divideRates(source.recommendedRate, target.recommendedRate);
    const recommendedRate = derivedRate ? compactRate(derivedRate) : null;
    return recommendedRate
        ? {
              ...source,
              settlementPreviewCurrency: targetCurrency,
              recommendedRate,
          }
        : null;
}

export function CurrencySettingsEditor({
    currencies,
    settings,
    onChange,
    disabled = false,
}: CurrencySettingsEditorProps) {
    const [currencyToAdd, setCurrencyToAdd] = useState("");
    const [recommendationResponse, setRecommendationResponse] =
        useState<RecommendationResponse | null>(null);
    const [recommendationError, setRecommendationError] = useState(false);
    const [recommendationsLoading, setRecommendationsLoading] = useState(false);
    const [refreshVersion, setRefreshVersion] = useState(0);
    const recommendationBaseRef = useRef("");
    const recommendationRequestsRef = useRef(new Map<string, Promise<RecommendationResponse>>());
    const [rateDirections, setRateDirections] = useState<Record<string, RateDirection>>({});
    const [inverseDrafts, setInverseDrafts] = useState<Record<string, string>>({});
    const rows = settings.currencies;
    const currencyCodes = useMemo(
        () => rows.map(({ currency }) => currency).sort(),
        [rows]
    );
    const targetCurrency = settings.settlementPreviewCurrency;
    const currencyKey = currencyCodes.join(",");
    const availableToAdd = useMemo(
        () => currencies.filter(({ code }) => !currencyCodes.includes(code)),
        [currencies, currencyCodes]
    );
    const firstAvailableCode = availableToAdd[0]?.code ?? "";

    useEffect(() => {
        setCurrencyToAdd(firstAvailableCode);
    }, [firstAvailableCode]);

    useEffect(() => {
        const selectedCodes = currencyKey ? currencyKey.split(",") : [];
        if (!targetCurrency || selectedCodes.length < 2) {
            setRecommendationError(false);
            return;
        }
        if (!recommendationBaseRef.current) {
            recommendationBaseRef.current = targetCurrency;
        }
        const baseCurrency = recommendationBaseRef.current;
        const responseBase = recommendationResponse?.recommendations[0]?.settlementPreviewCurrency;
        const cachedCodes = responseBase === baseCurrency
            ? new Set(recommendationResponse?.recommendations.map(({ sourceCurrency }) => sourceCurrency))
            : new Set<string>();
        const missingCodes = selectedCodes.filter((code) => !cachedCodes.has(code));
        if (missingCodes.length === 0) return;

        const requestedCodes = Array.from(new Set([baseCurrency, ...missingCodes])).sort();
        const requestKey = `${refreshVersion}|${baseCurrency}|${requestedCodes.join(",")}`;
        let request = recommendationRequestsRef.current.get(requestKey);
        if (!request) {
            request = fetchRateRecommendations(requestedCodes, baseCurrency);
            recommendationRequestsRef.current.set(requestKey, request);
        }

        let active = true;
        setRecommendationsLoading(true);
        setRecommendationError(false);
        void request
            .then((data) => {
                if (!active) return;
                setRecommendationResponse((current) => {
                    const currentBase = current?.recommendations[0]?.settlementPreviewCurrency;
                    if (!current || currentBase !== baseCurrency) return data;
                    const merged = new Map(
                        current.recommendations.map((item) => [item.sourceCurrency, item])
                    );
                    data.recommendations.forEach((item) => merged.set(item.sourceCurrency, item));
                    return { ...data, recommendations: Array.from(merged.values()) };
                });
            })
            .catch(() => {
                if (active) setRecommendationError(true);
            })
            .finally(() => {
                if (active) setRecommendationsLoading(false);
            });
        return () => {
            active = false;
        };
    }, [currencyKey, recommendationResponse, refreshVersion, targetCurrency]);

    const updateRows = (nextRows: GroupCurrencySetting[]) =>
        onChange({ ...settings, currencies: nextRows });

    const updateRate = (currency: string, previewRate: string | null) => {
        updateRows(
            rows.map((current) =>
                current.currency === currency
                    ? { ...current, previewRate }
                    : current
            )
        );
    };

    const setPreviewCurrency = (currency: string) => {
        if (currency === settings.settlementPreviewCurrency) return;
        onChange({
            settlementPreviewCurrency: currency,
            currencies: rows.map((row) => ({
                ...row,
                enabledForNewExpenses:
                    row.currency === currency || row.enabledForNewExpenses,
                previewRate: row.currency === currency ? "1" : null,
            })),
        });
        setRateDirections({});
        setInverseDrafts({});
    };

    const toggleEnabled = (row: GroupCurrencySetting, enabled: boolean) => {
        if (!enabled && row.currency === settings.settlementPreviewCurrency) return;
        updateRows(
            rows.map((current) =>
                current.currency === row.currency
                    ? { ...current, enabledForNewExpenses: enabled }
                    : current
            )
        );
    };

    const removeCurrency = (row: GroupCurrencySetting) => {
        if (
            row.historical ||
            row.hasCurrentBalance ||
            row.currency === settings.settlementPreviewCurrency
        ) {
            return;
        }
        updateRows(rows.filter(({ currency }) => currency !== row.currency));
        setRateDirections((current) => {
            const next = { ...current };
            delete next[row.currency];
            return next;
        });
        setInverseDrafts((current) => {
            const next = { ...current };
            delete next[row.currency];
            return next;
        });
    };

    const setRateDirection = (
        row: GroupCurrencySetting,
        direction: RateDirection
    ) => {
        setRateDirections((current) => ({
            ...current,
            [row.currency]: direction,
        }));
        if (direction === "preview-to-source") {
            setInverseDrafts((current) => ({
                ...current,
                [row.currency]: compactRate(reciprocalRate(row.previewRate) ?? "") ?? "",
            }));
        }
    };

    const editRate = (
        row: GroupCurrencySetting,
        direction: RateDirection,
        value: string
    ) => {
        if (direction === "source-to-preview") {
            updateRate(row.currency, value || null);
            return;
        }
        setInverseDrafts((current) => ({ ...current, [row.currency]: value }));
        updateRate(row.currency, reciprocalRate(value));
    };

    const applyRecommendation = (
        row: GroupCurrencySetting,
        direction: RateDirection,
        recommendation: Recommendation
    ) => {
        updateRate(row.currency, recommendation.recommendedRate);
        if (direction === "preview-to-source") {
            setInverseDrafts((current) => ({
                ...current,
                [row.currency]: compactRate(reciprocalRate(recommendation.recommendedRate) ?? "") ?? "",
            }));
        }
    };

    const refreshRecommendations = () => {
        recommendationBaseRef.current = settings.settlementPreviewCurrency;
        recommendationRequestsRef.current.clear();
        setRecommendationResponse(null);
        setRecommendationError(false);
        setRefreshVersion((version) => version + 1);
    };

    const addCurrency = () => {
        if (!currencyToAdd) return;
        updateRows([
            ...rows,
            {
                currency: currencyToAdd,
                enabledForNewExpenses: true,
                historical: false,
                hasCurrentBalance: false,
                previewRate:
                    currencyToAdd === settings.settlementPreviewCurrency
                        ? "1"
                        : null,
            },
        ]);
    };

    return (
        <div className="grid gap-6">
            <Field>
                <FieldLabel>Settlement preview currency</FieldLabel>
                <CurrencyPicker
                    value={settings.settlementPreviewCurrency}
                    currencies={currencies.filter(({ code }) =>
                        currencyCodes.includes(code)
                    )}
                    onChange={setPreviewCurrency}
                    disabled={disabled || rows.length === 0}
                />
                <FieldDescription>
                    Rates convert each original balance into this approximate summary.
                    Changing this currency resets direction-dependent rates.
                </FieldDescription>
            </Field>

            <fieldset className="grid gap-3">
                <legend className="text-sm font-semibold">Enabled and retained currencies</legend>
                {rows.map((row) => {
                    const recommendation = recommendationForCurrency(
                        recommendationResponse,
                        row.currency,
                        settings.settlementPreviewCurrency
                    );
                    const retained = row.historical || row.hasCurrentBalance;
                    const isPreview = row.currency === settings.settlementPreviewCurrency;
                    const direction = rateDirections[row.currency] ?? "source-to-preview";
                    const inverseDraft = inverseDrafts[row.currency] ?? compactRate(reciprocalRate(row.previewRate) ?? "") ?? "";
                    const displayedRate = direction === "source-to-preview"
                        ? row.previewRate ?? ""
                        : inverseDraft;
                    const displayedRecommendation = recommendation
                        ? direction === "source-to-preview"
                            ? recommendation.recommendedRate
                            : compactRate(reciprocalRate(recommendation.recommendedRate) ?? "")
                        : null;
                    const displayedRateIsValid = direction === "source-to-preview"
                        ? validPositiveRate(row.previewRate)
                        : validPositiveRate(displayedRate) && reciprocalRate(displayedRate) !== null;
                    const inputId = `preview-rate-${row.currency}`;
                    return (
                        <div key={row.currency} className="rounded-2xl border border-border bg-background p-4">
                            <div className="flex items-start gap-3">
                                <input
                                    type="checkbox"
                                    className="mt-0.5 size-4 shrink-0 accent-primary"
                                    id={`currency-enabled-${row.currency}`}
                                    checked={row.enabledForNewExpenses}
                                    disabled={disabled || isPreview}
                                    onChange={(event) =>
                                        toggleEnabled(row, event.target.checked)
                                    }
                                />
                                <label className="min-w-0 flex-1" htmlFor={`currency-enabled-${row.currency}`}>
                                    <span className="font-semibold">{row.currency}</span>
                                    <span className="ml-2 text-xs text-foreground/60">
                                        {row.enabledForNewExpenses ? "Enabled for new expenses" : "Historical only"}
                                    </span>
                                    {retained ? (
                                        <span className="mt-1 block text-xs text-foreground/60">
                                            Previously used or currently owed; it cannot be removed.
                                        </span>
                                    ) : isPreview ? (
                                        <span className="mt-1 block text-xs text-foreground/60">
                                            Choose another settlement preview currency before removing this one.
                                        </span>
                                    ) : null}
                                </label>
                                {!retained ? (
                                    <Button
                                        type="button"
                                        variant="destructive"
                                        size="icon"
                                        className="min-h-12 min-w-12"
                                        aria-label={`Remove ${row.currency}`}
                                        title={isPreview ? "Choose another settlement preview currency first" : `Remove ${row.currency}`}
                                        disabled={disabled || isPreview}
                                        onClick={() => removeCurrency(row)}
                                    >
                                        <Icon path={mdiTrashCanOutline} size={0.85} aria-hidden="true" />
                                    </Button>
                                ) : null}
                            </div>
                            {!isPreview ? (
                                <div className="mt-4" role="group" aria-label={`${row.currency} rate direction`}>
                                    <p className="mb-2 text-xs font-medium text-foreground/65">Rate direction</p>
                                    <div className="grid grid-cols-2 rounded-xl bg-muted p-1">
                                        <Button
                                            type="button"
                                            variant={direction === "source-to-preview" ? "secondary" : "ghost"}
                                            className="min-h-11"
                                            aria-pressed={direction === "source-to-preview"}
                                            disabled={disabled}
                                            onClick={() => setRateDirection(row, "source-to-preview")}
                                        >
                                            {row.currency} → {settings.settlementPreviewCurrency}
                                        </Button>
                                        <Button
                                            type="button"
                                            variant={direction === "preview-to-source" ? "secondary" : "ghost"}
                                            className="min-h-11"
                                            aria-pressed={direction === "preview-to-source"}
                                            disabled={disabled}
                                            onClick={() => setRateDirection(row, "preview-to-source")}
                                        >
                                            {settings.settlementPreviewCurrency} → {row.currency}
                                        </Button>
                                    </div>
                                </div>
                            ) : null}
                            <Field className="mt-4" data-invalid={!displayedRateIsValid}>
                                <FieldLabel htmlFor={inputId}>
                                    {direction === "source-to-preview" || isPreview
                                        ? `1 ${row.currency} = … ${settings.settlementPreviewCurrency}`
                                        : `1 ${settings.settlementPreviewCurrency} = … ${row.currency}`}
                                </FieldLabel>
                                <Input
                                    id={inputId}
                                    inputMode="decimal"
                                    value={displayedRate}
                                    disabled={disabled || isPreview}
                                    aria-invalid={!displayedRateIsValid}
                                    onChange={(event) =>
                                        editRate(row, direction, event.target.value)
                                    }
                                />
                                {!displayedRateIsValid ? (
                                    <FieldDescription role="alert" className="text-destructive">
                                        Enter a positive rate with at most 15 digits before and after the decimal.
                                    </FieldDescription>
                                ) : null}
                            </Field>
                            {recommendation && !isPreview && displayedRecommendation ? (
                                <div className="mt-3 flex flex-col gap-3 rounded-xl border border-primary/20 bg-primary/5 p-3 sm:flex-row sm:items-center sm:justify-between">
                                    <div className="min-w-0">
                                        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-primary">
                                            Recommended rate
                                        </p>
                                        <p className="mt-1 font-semibold text-foreground">
                                            {direction === "source-to-preview"
                                                ? `1 ${row.currency} = ${displayedRecommendation} ${settings.settlementPreviewCurrency}`
                                                : `1 ${settings.settlementPreviewCurrency} = ${displayedRecommendation} ${row.currency}`}
                                        </p>
                                        <p className="mt-1 text-xs text-foreground/60">
                                            <a
                                                className="font-medium underline underline-offset-2"
                                                href={recommendationResponse?.attributionUrl}
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                {recommendationResponse?.providerName}
                                            </a>{" "}
                                            published {recommendation.rateSnapshotAt}
                                        </p>
                                    </div>
                                    <Button
                                        type="button"
                                        className="min-h-12 w-full sm:w-auto sm:min-w-28"
                                        aria-label={`Apply recommended rate for ${row.currency}`}
                                        disabled={disabled}
                                        onClick={() =>
                                            applyRecommendation(row, direction, recommendation)
                                        }
                                    >
                                        Apply rate
                                    </Button>
                                </div>
                            ) : null}
                        </div>
                    );
                })}
            </fieldset>

            {availableToAdd.length > 0 ? (
                <div className="grid gap-3">
                    <Field>
                        <FieldLabel>Add currency</FieldLabel>
                        <CurrencyPicker
                            value={currencyToAdd}
                            currencies={availableToAdd}
                            onChange={setCurrencyToAdd}
                            disabled={disabled}
                        />
                    </Field>
                    <Button
                        type="button"
                        className="min-h-12 w-full whitespace-nowrap md:w-auto md:min-w-36 md:justify-self-end"
                        disabled={disabled || !currencyToAdd}
                        onClick={addCurrency}
                    >
                        Add currency
                    </Button>
                </div>
            ) : null}

            {currencyCodes.length > 1 ? (
                <div className="flex flex-wrap items-center gap-3 text-sm">
                    <Button
                        type="button"
                        variant="ghost"
                        disabled={disabled || recommendationsLoading}
                        onClick={refreshRecommendations}
                    >
                        <Icon data-icon="inline-start" path={mdiRefresh} size={0.8} aria-hidden="true" />
                        {recommendationsLoading ? "Loading recommendation…" : "Refresh recommendation"}
                    </Button>
                    {recommendationError ? (
                        <span className="text-foreground/60" role="status">
                            Recommendations are unavailable. You can still enter rates manually.
                        </span>
                    ) : null}
                </div>
            ) : null}
        </div>
    );
}
