const MONTH_NAMES = [
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

const REVIEW_MONTH_PATTERN = /^(\d{4})-(0[1-9]|1[0-2])$/;

export function parseReviewMonth(value: string) {
    const match = REVIEW_MONTH_PATTERN.exec(value);
    if (!match) return null;
    return { year: Number(match[1]), month: Number(match[2]) };
}

export function formatReviewMonth(value: string) {
    const parsed = parseReviewMonth(value);
    return parsed ? `${MONTH_NAMES[parsed.month - 1]} ${parsed.year}` : value;
}

export function shiftReviewMonth(value: string, delta: number) {
    const parsed = parseReviewMonth(value);
    if (!parsed) return value;
    const zeroBased = parsed.year * 12 + parsed.month - 1 + delta;
    const year = Math.floor(zeroBased / 12);
    const month = (zeroBased % 12 + 12) % 12 + 1;
    return `${year.toString().padStart(4, "0")}-${month.toString().padStart(2, "0")}`;
}

export function previousClosedUTCMonth(now = new Date()) {
    const current = `${now.getUTCFullYear().toString().padStart(4, "0")}-${(now.getUTCMonth() + 1).toString().padStart(2, "0")}`;
    return shiftReviewMonth(current, -1);
}
