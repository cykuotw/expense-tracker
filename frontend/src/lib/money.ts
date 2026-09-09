export interface CurrencyMetadata {
    code: string;
    displayName: string;
    minorUnitDigits: number;
    amountDigits: number;
}

export function currencyAmountDigits(
    currencies: CurrencyMetadata[],
    currencyCode: string
): number | null {
    const amountDigits = currencies.find(
        ({ code }) => code === currencyCode
    )?.amountDigits;
    return typeof amountDigits === "number" ? amountDigits : null;
}

export function moneyInputStep(amountDigits: number): string {
    return amountDigits === 0
        ? "1"
        : `0.${"0".repeat(amountDigits - 1)}1`;
}

export function moneyInputPlaceholder(amountDigits: number): string {
    return unitsToDecimal(0n, amountDigits);
}

const DECIMAL_PATTERN = /^(?:0|[1-9]\d*)(?:\.\d+)?$/;

export function decimalToUnits(value: string, amountDigits: number): bigint | null {
    if (!Number.isInteger(amountDigits) || amountDigits < 0 || amountDigits > 3) {
        return null;
    }
    if (!DECIMAL_PATTERN.test(value)) return null;

    const [whole, fraction = ""] = value.split(".");
    const significantFraction = fraction.replace(/0+$/, "");
    if (significantFraction.length > amountDigits) return null;

    const paddedFraction = fraction.padEnd(amountDigits, "0").slice(0, amountDigits);
    return BigInt(whole) * 10n ** BigInt(amountDigits) + BigInt(paddedFraction || "0");
}

export function unitsToDecimal(units: bigint, amountDigits: number): string {
    const negative = units < 0n;
    const absolute = negative ? -units : units;
    const scale = 10n ** BigInt(amountDigits);
    const whole = absolute / scale;
    const fraction = (absolute % scale).toString().padStart(amountDigits, "0");
    return `${negative ? "-" : ""}${whole}${amountDigits === 0 ? "" : `.${fraction}`}`;
}

export function normalizeMoney(value: string, amountDigits: number): string | null {
    const units = decimalToUnits(value, amountDigits);
    return units === null ? null : unitsToDecimal(units, amountDigits);
}

export function splitEqually(total: bigint, participantIDs: string[]): Map<string, bigint> | null {
    if (participantIDs.length === 0 || new Set(participantIDs).size !== participantIDs.length) {
        return null;
    }
    const sortedIDs = [...participantIDs].sort((left, right) => left.localeCompare(right));
    const base = total / BigInt(sortedIDs.length);
    const remainder = total % BigInt(sortedIDs.length);
    return new Map(sortedIDs.map((userID, index) => [userID, base + (BigInt(index) < remainder ? 1n : 0n)]));
}

export function formatMoney(value: string, amountDigits: number, currency: string): string {
    const normalized = normalizeMoney(value, amountDigits);
    return `${normalized ?? value} ${currency}`;
}
