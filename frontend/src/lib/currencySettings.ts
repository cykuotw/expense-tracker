import { GroupCurrencySettings } from "../types/group";

const RATE_INTEGER_DIGITS = 15;
const RATE_FRACTION_DIGITS = 15;
const DECIMAL_PARTS = /^(?:([0-9]+)(?:\.([0-9]+))?|\.([0-9]+))$/;

function parseRate(value: string) {
    const match = DECIMAL_PARTS.exec(value);
    if (!match) return null;

    const integer = match[1] ?? "0";
    const fraction = match[2] ?? match[3] ?? "";
    const significantInteger = integer.replace(/^0+/, "");
    if (
        significantInteger.length > RATE_INTEGER_DIGITS ||
        fraction.length > RATE_FRACTION_DIGITS
    ) {
        return null;
    }

    const digits = BigInt(`${integer}${fraction}`);
    if (digits <= 0n) return null;
    return { digits, fraction, integer, scale: fraction.length };
}

function formatScaledRate(scaled: bigint, scale: number): string | null {
    if (scaled <= 0n) return null;
    const padded = scaled.toString().padStart(scale + 1, "0");
    const integer = scale === 0 ? padded : padded.slice(0, -scale);
    if (integer.length > RATE_INTEGER_DIGITS) return null;
    if (scale === 0) return integer;
    const fraction = padded.slice(-scale).replace(/0+$/, "");
    return fraction ? `${integer}.${fraction}` : integer;
}

function divideAndRound(numerator: bigint, denominator: bigint, scale: number) {
    let quotient = numerator / denominator;
    const remainder = numerator % denominator;
    if (remainder * 2n >= denominator) quotient += 1n;
    return formatScaledRate(quotient, scale);
}

export function validPositiveRate(value: string | null): value is string {
    return value !== null && parseRate(value) !== null;
}

export function divideRates(dividend: string, divisor: string): string | null {
    const left = parseRate(dividend);
    const right = parseRate(divisor);
    if (!left || !right) return null;
    return divideAndRound(
        left.digits * 10n ** BigInt(right.scale + RATE_FRACTION_DIGITS),
        right.digits * 10n ** BigInt(left.scale),
        RATE_FRACTION_DIGITS
    );
}

/** Rounds a valid rate to a readable number of significant digits. */
export function compactRate(value: string, significantDigits = 8): string | null {
    const parsed = parseRate(value);
    if (!parsed || significantDigits < 1) return null;
    const significantIntegerDigits = parsed.integer.replace(/^0+/, "").length;
    const leadingFractionZeros = parsed.fraction.match(/^0*/)?.[0].length ?? 0;
    const desiredScale = Math.min(
        RATE_FRACTION_DIGITS,
        significantIntegerDigits > 0
            ? Math.max(0, significantDigits - significantIntegerDigits)
            : leadingFractionZeros + significantDigits
    );
    if (parsed.scale <= desiredScale) {
        return formatScaledRate(parsed.digits, parsed.scale);
    }
    return divideAndRound(
        parsed.digits,
        10n ** BigInt(parsed.scale - desiredScale),
        desiredScale
    );
}

/**
 * Returns a reciprocal rounded half-up to the persisted NUMERIC(30,15)
 * contract. Null means the input or its reciprocal cannot fit that contract.
 */
export function reciprocalRate(value: string | null): string | null {
    if (value === null) return null;
    const parsed = parseRate(value);
    if (!parsed) return null;
    return divideAndRound(
        10n ** BigInt(parsed.scale + RATE_FRACTION_DIGITS),
        parsed.digits,
        RATE_FRACTION_DIGITS
    );
}

export function currencySettingsAreValid(settings: GroupCurrencySettings) {
    const preview = settings.currencies.find(
        ({ currency }) => currency === settings.settlementPreviewCurrency
    );
    return Boolean(
        preview?.enabledForNewExpenses &&
            preview.previewRate === "1" &&
            settings.currencies.some(({ enabledForNewExpenses }) => enabledForNewExpenses) &&
            settings.currencies.every(({ previewRate }) => validPositiveRate(previewRate))
    );
}

export function legacyCurrencySettings(currency: string): GroupCurrencySettings {
    return {
        settlementPreviewCurrency: currency,
        currencies: currency
            ? [{
                  currency,
                  enabledForNewExpenses: true,
                  historical: false,
                  hasCurrentBalance: false,
                  previewRate: "1",
              }]
            : [],
    };
}
