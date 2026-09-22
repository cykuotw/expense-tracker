import { useCallback, useEffect, useRef, useState } from "react";
import { mdiChevronLeft, mdiChevronRight, mdiHistory, mdiReceiptTextOutline } from "@mdi/js";
import Icon from "@mdi/react";
import { Link, useNavigate, useParams } from "react-router-dom";
import DesktopBackLink from "../components/DesktopBackLink";
import MobilePageHeader from "../components/MobilePageHeader";
import { ReviewMonthPicker } from "../components/review/ReviewMonthPicker";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { useCurrencies } from "../hooks/useCurrencies";
import { getExpenseTypePresentation } from "../lib/expenseCategoryPresentation";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { formatDateOnlyLong } from "../lib/dateOnly";
import { currencyAmountDigits, formatMoney } from "../lib/money";
import { formatReviewMonth, parseReviewMonth, previousClosedUTCMonth, shiftReviewMonth } from "../lib/reviewMonth";
import { cn } from "../lib/utils";
import type { MonthlyReviewCurrency, MonthlyReviewData, MonthlyReviewExpense, MonthlyReviewExpensePage, MonthlyReviewTrendCurrency, MonthlyReviewTrendData } from "../types/monthlyReview";

const EARLIEST_REVIEW_MONTH = "2026-01";
const MOBILE_TREND_MONTHS = 6;

function formatSignedMoney(value: string, currency: string, amountDigits: number | null) {
    if (amountDigits === null) return `${value} ${currency}`;
    if (!value.startsWith("-")) return formatMoney(value, amountDigits, currency);
    return `-${formatMoney(value.slice(1), amountDigits, currency)}`;
}

function numericAmount(value: string) {
    const amount = Number(value);
    return Number.isFinite(amount) ? amount : 0;
}

function formatChartAmount(value: string, amountDigits: number | null) {
    const amount = numericAmount(value);
    return new Intl.NumberFormat(undefined, {
        maximumFractionDigits: amountDigits ?? 2,
        notation: Math.abs(amount) >= 10_000 ? "compact" : "standard",
    }).format(amount);
}

function ReviewState({ review }: { review: MonthlyReviewData }) {
    if (review.state === "published") return null;
    const content = {
        not_closed: { title: "This month is still open", description: "The review becomes available after the month closes in UTC." },
        unpublished: { title: "Review publication is pending", description: "The daily publisher will make this closed month available automatically." },
        empty: { title: "No expenses for this month", description: "Choose another month or add an expense dated in this month." },
    }[review.state];
    return (
        <section className="panel-card flex flex-col gap-4 rounded-[1.5rem] p-5 sm:p-6">
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary" aria-hidden="true">
                <Icon path={mdiReceiptTextOutline} size={1.1} />
            </div>
            <div className="flex flex-col gap-1">
                <h2 className="section-title">{content.title}</h2>
                <p className="text-sm leading-6 text-foreground/65">{content.description}</p>
            </div>
        </section>
    );
}

function HorizontalBarCard({ title, rows, currency, amountDigits }: {
    title: string;
    rows: Array<{ key: string; label: string; amount: string }>;
    currency: string;
    amountDigits: number | null;
}) {
    const total = rows.reduce((sum, row) => sum + Math.max(0, numericAmount(row.amount)), 0);
    return (
        <Card className="panel-card rounded-[1.5rem]" size="sm">
            <CardHeader><CardTitle><h3>{title}</h3></CardTitle></CardHeader>
            <CardContent>
                <div className="flex flex-col gap-4">
                    {rows.map((row) => {
                        const amount = Math.max(0, numericAmount(row.amount));
                        const proportion = total > 0 ? amount / total : 0;
                        const percentage = Math.round(proportion * 100);
                        return (
                            <div key={row.key} className="flex flex-col gap-2">
                                <div className="flex min-w-0 items-baseline justify-between gap-3">
                                    <span className="truncate text-sm font-medium">{row.label}</span>
                                    <span className="shrink-0 text-sm font-semibold tabular-nums">{formatSignedMoney(row.amount, currency, amountDigits)}</span>
                                </div>
                                <div className="h-2.5 overflow-hidden rounded-full bg-primary/10" role="img" aria-label={`${row.label}: ${percentage}%`}>
                                    <div className="h-full rounded-full bg-primary" style={{ width: `${proportion * 100}%` }} />
                                </div>
                                <span className="text-xs text-foreground/55">{percentage}% of total</span>
                            </div>
                        );
                    })}
                </div>
            </CardContent>
        </Card>
    );
}

function MemberNetCard({ rows, currency, amountDigits }: {
    rows: MonthlyReviewCurrency["memberNetTotals"];
    currency: string;
    amountDigits: number | null;
}) {
    return (
        <Card className="panel-card rounded-[1.5rem]" size="sm">
            <CardHeader><CardTitle><h3>Member net totals</h3></CardTitle></CardHeader>
            <CardContent>
                <div className="flex flex-col gap-2">
                    {rows.map((row) => (
                        <div key={row.userId} className="metric-card flex min-h-12 items-center justify-between gap-3 rounded-xl px-3 py-2">
                            <span className="min-w-0 truncate font-medium">{row.username}</span>
                            <span className="shrink-0 font-semibold tabular-nums">{formatSignedMoney(row.amount, currency, amountDigits)}</span>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}

function ExpenseDisclosure({ summary, amountDigits, groupId, month }: { summary: MonthlyReviewCurrency; amountDigits: number | null; groupId: string; month: string }) {
    const [expenses, setExpenses] = useState<MonthlyReviewExpense[]>([]);
    const [nextCursor, setNextCursor] = useState<string | undefined>();
    const [loaded, setLoaded] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const requestInFlight = useRef(false);

    const loadExpenses = useCallback(async (cursor?: string) => {
        if (requestInFlight.current) return;
        requestInFlight.current = true;
        setLoading(true);
        setError("");
        try {
            const query = new URLSearchParams({ currency: summary.currency });
            if (cursor) query.set("cursor", cursor);
            const response = await apiFetch(`/group/${encodeURIComponent(groupId)}/monthly-review/${month}/expenses?${query.toString()}`);
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Expenses could not be loaded."));
            const page = (await response.json()) as MonthlyReviewExpensePage;
            setExpenses((current) => cursor ? [...current, ...asArray<MonthlyReviewExpense>(page.expenses)] : asArray<MonthlyReviewExpense>(page.expenses));
            setNextCursor(page.nextCursor);
            setLoaded(true);
        } catch (loadError) {
            setError(loadError instanceof Error ? loadError.message : "Expenses could not be loaded.");
        } finally {
            requestInFlight.current = false;
            setLoading(false);
        }
    }, [groupId, month, summary.currency]);

    return (
        <details
            className="panel-card group rounded-[1.5rem] p-4 sm:p-5"
            onToggle={(event) => {
                if (event.currentTarget.open && !loaded && !requestInFlight.current) void loadExpenses();
            }}
        >
            <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary [&::-webkit-details-marker]:hidden">
                <span className="font-semibold">Expenses</span>
                <span className="flex items-center gap-2 text-sm text-foreground/60">
                    {summary.expenseCount}
                    <Icon className="transition-transform group-open:rotate-90" path={mdiChevronRight} size={0.8} aria-hidden="true" />
                </span>
            </summary>
            <div className="mt-3 flex flex-col gap-2">
                {expenses.map((expense) => {
                    const presentation = getExpenseTypePresentation(expense.category, expense.expenseType);
                    return (
                        <Link key={expense.id} to={`/expense/${expense.id}`} className="metric-card grid min-h-16 grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-3 rounded-2xl p-3 transition-colors hover:border-primary/30 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary" aria-label={`Open expense: ${expense.description}`}>
                            <span className={`flex size-10 items-center justify-center rounded-xl ${presentation.iconClassName}`} aria-hidden="true"><Icon path={presentation.icon} size={1} /></span>
                            <span className="min-w-0">
                                <span className="block truncate font-semibold">{expense.description}</span>
                                <span className="block truncate text-xs text-foreground/60">{formatDateOnlyLong(expense.occurredOn)} · Paid by {expense.payerName}</span>
                            </span>
                            <span className="flex flex-col items-end gap-1">
                                <span className="font-semibold tabular-nums">{formatSignedMoney(expense.total, summary.currency, amountDigits)}</span>
                                {expense.settled ? <Badge variant="secondary">Settled</Badge> : null}
                            </span>
                        </Link>
                    );
                })}
                {loading ? <div className="flex min-h-20 items-center justify-center" role="status"><span className="ui-spinner" aria-hidden="true" /><span className="sr-only">Loading expenses</span></div> : null}
                {!loading && loaded && expenses.length === 0 ? <p className="py-4 text-center text-sm text-foreground/60">No expenses available.</p> : null}
                {error ? <div className="flex flex-col items-center gap-2 py-3" role="alert"><p className="text-sm text-destructive">{error}</p><Button type="button" variant="outline" onClick={() => void loadExpenses(nextCursor)}>Try again</Button></div> : null}
                {!loading && !error && nextCursor ? <Button type="button" variant="outline" onClick={() => void loadExpenses(nextCursor)}>Load more expenses</Button> : null}
            </div>
        </details>
    );
}

function CurrencyReview({ summary, amountDigits, groupId, month }: { summary: MonthlyReviewCurrency; amountDigits: number | null; groupId: string; month: string }) {
    return (
        <section className="flex flex-col gap-4" aria-labelledby={`currency-${summary.currency}`}>
            <Card className="panel-card rounded-[1.75rem]">
                <CardHeader><CardTitle><h2 id={`currency-${summary.currency}`}>{summary.currency} spending</h2></CardTitle><CardDescription>{summary.expenseCount} expense{summary.expenseCount === 1 ? "" : "s"}</CardDescription></CardHeader>
                <CardContent><div className="text-3xl font-bold tracking-[-0.04em] text-primary stat-number">{formatSignedMoney(summary.total, summary.currency, amountDigits)}</div></CardContent>
            </Card>
            <div className="grid gap-4 lg:grid-cols-2">
                <HorizontalBarCard title="By category" rows={summary.categories.map((item) => ({ key: item.name, label: item.name, amount: item.amount }))} currency={summary.currency} amountDigits={amountDigits} />
                <HorizontalBarCard title="Paid by" rows={summary.payers.map((item) => ({ key: item.userId, label: item.username, amount: item.amount }))} currency={summary.currency} amountDigits={amountDigits} />
            </div>
            <MemberNetCard rows={summary.memberNetTotals} currency={summary.currency} amountDigits={amountDigits} />
            <ExpenseDisclosure summary={summary} amountDigits={amountDigits} groupId={groupId} month={month} />
        </section>
    );
}

function mobileTrendBounds(monthCount: number, page: number) {
    const start = Math.max(0, monthCount - (page + 1) * MOBILE_TREND_MONTHS);
    return { start, end: Math.min(monthCount, start + MOBILE_TREND_MONTHS) };
}

function TrendChart({ summary, amountDigits, groupId, mobilePage }: { summary: MonthlyReviewTrendCurrency; amountDigits: number | null; groupId: string; mobilePage: number }) {
    const maximum = Math.max(0, ...summary.months.map((item) => numericAmount(item.total)));
    const mobileBounds = mobileTrendBounds(summary.months.length, mobilePage);
    return (
        <Card className="panel-card rounded-[1.5rem]" size="sm">
            <CardHeader><CardTitle><h3>{summary.currency} trend</h3></CardTitle></CardHeader>
            <CardContent>
                <div className="grid h-48 grid-cols-6 items-end gap-2 md:grid-cols-12" role="group" aria-label={`${summary.currency} monthly spending trend`}>
                    {summary.months.map((item, index) => {
                        const amount = numericAmount(item.total);
                        const height = maximum > 0 && amount > 0 ? Math.max(6, amount / maximum * 100) : 0;
                        const visibleOnMobile = index >= mobileBounds.start && index < mobileBounds.end;
                        return (
                            <div key={item.month} className={cn("h-full min-w-0 flex-col items-center justify-end gap-2", visibleOnMobile ? "flex" : "hidden md:flex")}>
                                <span
                                    className="max-w-full truncate rounded-full bg-primary/10 px-1.5 py-0.5 text-center text-[0.65rem] font-semibold tabular-nums text-foreground/75 sm:text-xs"
                                    title={formatSignedMoney(item.total, summary.currency, amountDigits)}
                                >
                                    {formatChartAmount(item.total, amountDigits)}
                                </span>
                                <Link to={`/group/${groupId}/monthly-review/${item.month}`} className="flex w-full flex-1 items-end rounded-lg bg-primary/5 p-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary" aria-label={`${formatReviewMonth(item.month)}: ${formatSignedMoney(item.total, summary.currency, amountDigits)}, ${item.expenseCount} expenses`}>
                                    <span className="block w-full rounded-md bg-primary transition-colors hover:bg-primary/80" style={{ height: `${height}%` }} />
                                </Link>
                                <span className="text-center text-[0.65rem] font-semibold text-foreground/60 sm:text-xs">{formatReviewMonth(item.month).slice(0, 3)}</span>
                            </div>
                        );
                    })}
                </div>
            </CardContent>
        </Card>
    );
}

function TrendSection({ trend, loading, error, amountDigitsFor, onPrevious, onNext, onLatest }: {
    trend: MonthlyReviewTrendData | null;
    loading: boolean;
    error: string;
    amountDigitsFor: (currency: string) => number | null;
    onPrevious: () => void;
    onNext: () => void;
    onLatest: () => void;
}) {
    const [mobilePage, setMobilePage] = useState(0);
    const trendMonths = trend?.currencies[0]?.months ?? [];
    const mobilePageCount = Math.max(1, Math.ceil(trendMonths.length / MOBILE_TREND_MONTHS));
    const mobileBounds = mobileTrendBounds(trendMonths.length, mobilePage);
    const mobileStartMonth = trendMonths[mobileBounds.start]?.month ?? trend?.startMonth;
    const mobileEndMonth = trendMonths[Math.max(mobileBounds.start, mobileBounds.end - 1)]?.month ?? trend?.endMonth;

    const previousMobileRange = () => {
        if (mobilePage < mobilePageCount - 1) {
            setMobilePage((current) => current + 1);
            return;
        }
        onPrevious();
    };
    const nextMobileRange = () => {
        if (mobilePage > 0) {
            setMobilePage((current) => current - 1);
            return;
        }
        onNext();
    };
    const latestMobileRange = () => {
        setMobilePage(0);
        onLatest();
    };

    return (
        <section className="flex flex-col gap-3" aria-labelledby="spending-trend-title">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <div className="section-label">History</div>
                    <h2 className="section-title mt-1" id="spending-trend-title">Spending history</h2>
                    {trend ? <p className="mt-1 hidden text-sm text-foreground/60 md:block">{formatReviewMonth(trend.startMonth)}–{formatReviewMonth(trend.endMonth)}</p> : null}
                    {mobileStartMonth && mobileEndMonth ? <p className="mt-1 text-sm text-foreground/60 md:hidden">{formatReviewMonth(mobileStartMonth)}–{formatReviewMonth(mobileEndMonth)}</p> : null}
                </div>
                <div className="grid grid-cols-3 gap-2 md:hidden">
                    <button className="ui-button ui-button-outline min-h-11 px-2 text-xs" type="button" aria-label="Previous six months" disabled={!trend || (mobilePage >= mobilePageCount - 1 && trend.startMonth <= EARLIEST_REVIEW_MONTH)} onClick={previousMobileRange}><Icon path={mdiChevronLeft} size={0.8} aria-hidden="true" /><span>Prev 6</span></button>
                    <button className="ui-button ui-button-ghost min-h-11 px-2 text-xs" type="button" aria-label="Latest six months" onClick={latestMobileRange}><Icon path={mdiHistory} size={0.8} aria-hidden="true" /><span>Latest</span></button>
                    <button className="ui-button ui-button-outline min-h-11 px-2 text-xs" type="button" aria-label="Next six months" disabled={!trend || (mobilePage === 0 && trend.endMonth >= previousClosedUTCMonth())} onClick={nextMobileRange}><span>Next 6</span><Icon path={mdiChevronRight} size={0.8} aria-hidden="true" /></button>
                </div>
                <div className="hidden grid-cols-3 gap-2 md:grid">
                    <button className="ui-button ui-button-outline min-h-11 px-3" type="button" aria-label="Previous twelve months" disabled={!trend || trend.startMonth <= EARLIEST_REVIEW_MONTH} onClick={onPrevious}><Icon path={mdiChevronLeft} size={0.8} aria-hidden="true" /><span>Previous 12</span></button>
                    <button className="ui-button ui-button-ghost min-h-11 px-3" type="button" aria-label="Latest twelve months" onClick={onLatest}><Icon path={mdiHistory} size={0.8} aria-hidden="true" /><span>Latest</span></button>
                    <button className="ui-button ui-button-outline min-h-11 px-3" type="button" aria-label="Next twelve months" disabled={!trend || trend.endMonth >= previousClosedUTCMonth()} onClick={onNext}><span>Next 12</span><Icon path={mdiChevronRight} size={0.8} aria-hidden="true" /></button>
                </div>
            </div>
            {loading ? (
                <div className="panel-card flex min-h-48 items-center justify-center rounded-[1.5rem]" role="status"><span className="ui-spinner ui-spinner-lg" aria-hidden="true" /><span className="sr-only">Loading spending trend</span></div>
            ) : error ? (
                <div className="metric-card rounded-2xl p-4 text-sm text-foreground/65">{error}</div>
            ) : trend && trend.currencies.length > 0 ? (
                <div className="grid gap-4">{trend.currencies.map((summary) => <TrendChart key={summary.currency} summary={summary} amountDigits={amountDigitsFor(summary.currency)} groupId={trend.groupId} mobilePage={mobilePage} />)}</div>
            ) : (
                <div className="metric-card rounded-2xl p-4 text-sm text-foreground/65">No published spending in this range.</div>
            )}
        </section>
    );
}

export default function MonthlyReview() {
    const { id = "", month = "" } = useParams();
    const navigate = useNavigate();
    const latestClosedMonth = previousClosedUTCMonth();
    const [review, setReview] = useState<MonthlyReviewData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [trendEndMonth, setTrendEndMonth] = useState(latestClosedMonth);
    const [trend, setTrend] = useState<MonthlyReviewTrendData | null>(null);
    const [trendLoading, setTrendLoading] = useState(true);
    const [trendError, setTrendError] = useState("");
    const { currencies } = useCurrencies();
    const validMonth = parseReviewMonth(month) !== null;
    const amountDigitsFor = useCallback((currency: string) => currencyAmountDigits(currencies, currency), [currencies]);

    const load = useCallback(async (signal?: AbortSignal) => {
        if (!id || !validMonth) { setError("Choose a valid review month."); setLoading(false); return; }
        setLoading(true);
        setError("");
        try {
            const response = await apiFetch(
                `/group/${encodeURIComponent(id)}/monthly-review/${month}`,
                { signal },
            );
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Monthly review could not be loaded."));
            const data = (await response.json()) as MonthlyReviewData;
            if (signal?.aborted) return;
            setReview({ ...data, currencies: asArray<MonthlyReviewCurrency>(data.currencies) });
        } catch (loadError) {
            if (signal?.aborted) return;
            setReview(null);
            setError(loadError instanceof Error ? loadError.message : "Monthly review could not be loaded.");
        } finally {
            if (!signal?.aborted) setLoading(false);
        }
    }, [id, month, validMonth]);

    const loadTrend = useCallback(async (signal?: AbortSignal) => {
        if (!id) return;
        setTrendLoading(true);
        setTrendError("");
        try {
            const response = await apiFetch(
                `/group/${encodeURIComponent(id)}/monthly-review-trend/${trendEndMonth}`,
                { signal },
            );
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Spending trend could not be loaded."));
            const data = (await response.json()) as MonthlyReviewTrendData;
            if (signal?.aborted) return;
            setTrend({ ...data, currencies: asArray<MonthlyReviewTrendCurrency>(data.currencies) });
        } catch (loadError) {
            if (signal?.aborted) return;
            setTrend(null);
            setTrendError(loadError instanceof Error ? loadError.message : "Spending trend could not be loaded.");
        } finally {
            if (!signal?.aborted) setTrendLoading(false);
        }
    }, [id, trendEndMonth]);

    useEffect(() => {
        const abortController = new AbortController();
        void load(abortController.signal);
        return () => abortController.abort();
    }, [load]);
    useEffect(() => {
        const abortController = new AbortController();
        void loadTrend(abortController.signal);
        return () => abortController.abort();
    }, [loadTrend]);

    const goToMonth = (delta: number) => navigate(`/group/${id}/monthly-review/${shiftReviewMonth(month, delta)}`);
    const latestReportMonth = trend?.latestReportMonth;

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container flex flex-col gap-6">
                <MobilePageHeader title={review?.groupName ?? trend?.groupName ?? "Monthly review"} backTo={`/group/${id}`} backLabel="Back to group" />
                <DesktopBackLink to={`/group/${id}`} label="Back to group" />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy"><div className="page-eyebrow">Monthly review</div><h1 className="page-title">{review?.groupName ?? trend?.groupName ?? "Expense review"}</h1><p className="page-copy">Spending trends and live details from authoritative expenses.</p></div>
                </div>

                <TrendSection key={trendEndMonth} trend={trend} loading={trendLoading} error={trendError} amountDigitsFor={amountDigitsFor} onPrevious={() => setTrendEndMonth((current) => shiftReviewMonth(current, -12))} onNext={() => setTrendEndMonth((current) => { const next = shiftReviewMonth(current, 12); return next > latestClosedMonth ? latestClosedMonth : next; })} onLatest={() => setTrendEndMonth(latestClosedMonth)} />

                <section className="flex flex-col gap-3" aria-labelledby="review-detail-title">
                    <div><div className="section-label">Review detail</div><h2 className="section-title mt-1" id="review-detail-title">{formatReviewMonth(month)}</h2></div>
                    <nav className="panel-card-soft mx-auto flex w-full max-w-3xl flex-col gap-2 rounded-[1.5rem] p-3 sm:p-4" aria-label="Review month navigation">
                        <ReviewMonthPicker value={validMonth ? month : ""} minimumMonth={EARLIEST_REVIEW_MONTH} maximumMonth={latestClosedMonth} onChange={(nextMonth) => navigate(`/group/${id}/monthly-review/${nextMonth}`)} />
                        <div className="grid grid-cols-3 gap-2">
                            <button className="ui-button ui-button-outline min-h-11 px-2" disabled={!validMonth || month <= EARLIEST_REVIEW_MONTH} type="button" onClick={() => goToMonth(-1)}><Icon path={mdiChevronLeft} size={0.8} aria-hidden="true" /><span className="hidden sm:inline">Previous</span><span className="sm:hidden">Prev</span></button>
                            <button className="ui-button ui-button-primary min-h-11 px-2" aria-label="Latest report" disabled={!latestReportMonth || month === latestReportMonth} type="button" onClick={() => latestReportMonth && navigate(`/group/${id}/monthly-review/${latestReportMonth}`)}><span className="sm:hidden">Latest</span><span className="hidden sm:inline">Latest report</span></button>
                            <button className="ui-button ui-button-outline min-h-11 px-2" disabled={!validMonth || month >= latestClosedMonth} type="button" onClick={() => goToMonth(1)}><span>Next</span><Icon path={mdiChevronRight} size={0.8} aria-hidden="true" /></button>
                        </div>
                    </nav>

                    {loading ? (
                        <div className="flex min-h-48 items-center justify-center" role="status"><span className="ui-spinner ui-spinner-lg" aria-hidden="true" /><span className="sr-only">Loading monthly review</span></div>
                    ) : error ? (
                        <Card className="panel-card rounded-[1.5rem]"><CardHeader><CardTitle><h2>Monthly review unavailable</h2></CardTitle><CardDescription>{error}</CardDescription></CardHeader><CardContent><Button type="button" variant="outline" onClick={() => void load()}>Try again</Button></CardContent></Card>
                    ) : review && review.state !== "published" ? (
                        <ReviewState review={review} />
                    ) : review ? (
                        <div className="flex flex-col gap-8">{review.currencies.map((summary) => <CurrencyReview key={`${review.month}-${summary.currency}`} summary={summary} amountDigits={amountDigitsFor(summary.currency)} groupId={id} month={review.month} />)}</div>
                    ) : null}
                </section>
            </div>
        </div>
    );
}
