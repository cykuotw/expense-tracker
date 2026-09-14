import { useEffect, useMemo, useState } from "react";
import {
    mdiCalendarMonthOutline,
    mdiCheck,
    mdiChevronDown,
} from "@mdi/js";
import Icon from "@mdi/react";
import { formatReviewMonth, parseReviewMonth } from "../../lib/reviewMonth";
import { cn } from "../../lib/utils";

const MONTHS = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
] as const;

const EARLIEST_REVIEW_YEAR = 2026;

interface ReviewMonthPickerProps {
    value: string;
    onChange: (value: string) => void;
    minimumMonth?: string;
    maximumMonth?: string;
}

export function ReviewMonthPicker({
    value,
    onChange,
    minimumMonth = `${EARLIEST_REVIEW_YEAR}-01`,
    maximumMonth,
}: ReviewMonthPickerProps) {
    const selected = parseReviewMonth(value);
    const selectedYear = selected?.year;
    const currentYear = new Date().getUTCFullYear();
    const minimum = parseReviewMonth(minimumMonth) ?? { year: EARLIEST_REVIEW_YEAR, month: 1 };
    const maximum = parseReviewMonth(maximumMonth ?? `${currentYear}-12`) ?? {
        year: currentYear,
        month: 12,
    };
    const [open, setOpen] = useState(false);
    const [yearOpen, setYearOpen] = useState(false);
    const [displayYear, setDisplayYear] = useState(
        Math.min(maximum.year, Math.max(minimum.year, selectedYear ?? currentYear)),
    );
    const years = useMemo(() => {
        return Array.from(
            { length: Math.max(0, maximum.year - minimum.year + 1) },
            (_, index) => maximum.year - index,
        );
    }, [maximum.year, minimum.year]);

    useEffect(() => {
        if (selectedYear !== undefined) {
            setDisplayYear(Math.min(maximum.year, Math.max(minimum.year, selectedYear)));
        }
    }, [maximum.year, minimum.year, selectedYear]);

    return (
        <div
            className="relative min-w-0 flex-1"
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) {
                    setOpen(false);
                    setYearOpen(false);
                }
            }}
            onKeyDown={(event) => {
                if (event.key === "Escape") {
                    setOpen(false);
                    setYearOpen(false);
                }
            }}
        >
            <button
                type="button"
                className="flex min-h-14 w-full touch-manipulation items-center gap-3 rounded-2xl border border-border bg-background px-4 py-2 text-left transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                aria-label="Review month"
                aria-expanded={open}
                aria-haspopup="dialog"
                onClick={() => {
                    setOpen((current) => !current);
                    setYearOpen(false);
                }}
            >
                <span
                    className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary"
                    aria-hidden="true"
                >
                    <Icon path={mdiCalendarMonthOutline} size={1} />
                </span>
                <span className="min-w-0 flex-1">
                    <span className="block text-xs font-medium text-foreground/60">
                        Review month
                    </span>
                    <span className="block truncate font-semibold">
                        {selected ? formatReviewMonth(value) : "Choose a month"}
                    </span>
                </span>
                <Icon path={mdiChevronDown} size={0.9} aria-hidden="true" />
            </button>

            {open ? (
                <div
                    role="dialog"
                    aria-label="Choose review month"
                    className="absolute left-0 right-0 z-[60] mt-2 rounded-2xl border border-border bg-background p-3 shadow-xl sm:right-auto sm:w-80"
                >
                    <div className="flex items-center justify-between gap-3">
                        <span className="text-sm font-medium">Year</span>
                        <div className="relative w-32">
                            <button
                                type="button"
                                className="flex min-h-11 w-full items-center justify-between rounded-xl border border-border bg-background px-3 text-sm font-semibold transition hover:border-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                                aria-label="Review year"
                                aria-expanded={yearOpen}
                                aria-haspopup="listbox"
                                onClick={() => setYearOpen((current) => !current)}
                            >
                                <span>{displayYear}</span>
                                <Icon path={mdiChevronDown} size={0.8} aria-hidden="true" />
                            </button>
                            {yearOpen ? (
                                <div
                                    role="listbox"
                                    aria-label="Review year options"
                                    className="absolute right-0 z-[70] mt-2 max-h-48 w-full overflow-y-auto rounded-xl border border-border bg-background p-1.5 shadow-xl"
                                >
                                    {years.map((year) => (
                                        <button
                                            key={year}
                                            type="button"
                                            role="option"
                                            aria-selected={year === displayYear}
                                            className={cn(
                                                "flex min-h-11 w-full items-center justify-between rounded-lg px-3 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                                                year === displayYear && "bg-primary/10 text-primary",
                                            )}
                                            onClick={() => {
                                                setDisplayYear(year);
                                                setYearOpen(false);
                                            }}
                                        >
                                            <span>{year}</span>
                                            {year === displayYear ? (
                                                <Icon path={mdiCheck} size={0.65} aria-hidden="true" />
                                            ) : null}
                                        </button>
                                    ))}
                                </div>
                            ) : null}
                        </div>
                    </div>
                    <div className="mt-3 grid grid-cols-3 gap-2" role="group" aria-label={`${displayYear} months`}>
                        {MONTHS.map((monthName, index) => {
                            const monthNumber = index + 1;
                            const optionValue = `${displayYear.toString().padStart(4, "0")}-${monthNumber.toString().padStart(2, "0")}`;
                            const isSelected = optionValue === value;
                            const isDisabled = optionValue < minimumMonth || optionValue > (maximumMonth ?? `${currentYear}-12`);
                            return (
                                <button
                                    key={monthName}
                                    type="button"
                                    className={cn(
                                        "flex min-h-11 touch-manipulation items-center justify-center gap-1 rounded-xl px-2 text-sm font-medium transition hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                                        isSelected && "bg-primary/10 text-primary",
                                        isDisabled && "cursor-not-allowed opacity-40 hover:bg-transparent",
                                    )}
                                    aria-label={`${monthName} ${displayYear}`}
                                    aria-pressed={isSelected}
                                    disabled={isDisabled}
                                    onClick={() => {
                                        onChange(optionValue);
                                        setOpen(false);
                                        setYearOpen(false);
                                    }}
                                >
                                    <span>{monthName.slice(0, 3)}</span>
                                    {isSelected ? (
                                        <Icon path={mdiCheck} size={0.65} aria-hidden="true" />
                                    ) : null}
                                </button>
                            );
                        })}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
