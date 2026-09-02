const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;
const SHORT_MONTHS = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
] as const;

export interface DateOnlyParts {
    year: number;
    month: number;
    day: number;
}

export function parseDateOnly(value: string): DateOnlyParts | null {
    const match = DATE_ONLY_PATTERN.exec(value);
    if (!match) return null;

    const year = Number(match[1]);
    const month = Number(match[2]);
    const day = Number(match[3]);
    if (month < 1 || month > 12) return null;

    const leapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
    const daysInMonth = [31, leapYear ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    if (day < 1 || day > daysInMonth[month - 1]) return null;

    return { year, month, day };
}

export function isDateOnly(value: string): boolean {
    return parseDateOnly(value) !== null;
}

export function todayDateOnly(now = new Date()): string {
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, "0");
    const day = String(now.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
}

export function formatDateOnlyBrief(value: string): { month: string; day: string } {
    const parts = parseDateOnly(value);
    if (!parts) return { month: "", day: value };
    return { month: SHORT_MONTHS[parts.month - 1], day: String(parts.day) };
}

export function formatDateOnlyLong(value: string): string {
    const parts = parseDateOnly(value);
    if (!parts) return value;
    return `${SHORT_MONTHS[parts.month - 1]} ${parts.day}, ${parts.year}`;
}
