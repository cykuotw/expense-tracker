import { unitsToDecimal } from "./money";

interface Fraction { numerator: bigint; denominator: bigint }
export interface CalculatorResult { amount: string | null; error: string | null; rounded: boolean }

// Ignore one unfinished binary operation without masking malformed operator sequences.
export function normalizeCalculatorExpression(expression: string): string {
    return expression.replace(/([0-9)]|\d\.)\s*[+\-*/×÷−]\s*$/, "$1").trim();
}

function fraction(numerator: bigint, denominator: bigint): Fraction {
    if (denominator === 0n) throw new Error("Cannot divide by zero.");
    if (denominator < 0n) { numerator = -numerator; denominator = -denominator; }
    let left = numerator < 0n ? -numerator : numerator;
    let right = denominator;
    while (right) [left, right] = [right, left % right];
    return { numerator: numerator / left, denominator: denominator / left };
}

export function calculateAmount(expression: string, amountDigits: number | null): CalculatorResult {
    const invalid = (error: string | null): CalculatorResult => ({ amount: null, error, rounded: false });
    if (amountDigits === null || !Number.isInteger(amountDigits) || amountDigits < 0 || amountDigits > 3) {
        return invalid("Choose a currency first.");
    }
    if (!expression.trim()) return invalid(null);
    if (expression.length > 256) return invalid("Use a shorter calculation (up to 256 characters).");
    try {
        // A bounded arithmetic parser keeps decimal operations exact without executing input.
        const source = normalizeCalculatorExpression(expression).replace(/×/g, "*").replace(/÷/g, "/").replace(/−/g, "-");
        const tokenPattern = /\s*(\d+(?:\.\d*)?|\.\d+|[()+\-*/])/y;
        const tokens: string[] = [];
        let offset = 0;
        while (offset < source.length) {
            tokenPattern.lastIndex = offset;
            const match = tokenPattern.exec(source);
            if (!match) throw new Error("Enter a valid calculation using numbers and +, −, ×, ÷.");
            tokens.push(match[1]);
            offset = tokenPattern.lastIndex;
        }
        let index = 0;
        const primary = (): Fraction => {
            const token = tokens[index++];
            if (token === "+") return primary();
            if (token === "-") {
                const value = primary();
                return { ...value, numerator: -value.numerator };
            }
            if (token === "(") {
                const value = sum();
                if (tokens[index++] !== ")") throw new Error("Close the parentheses to finish the calculation.");
                return value;
            }
            if (!token || !/^(?:\d+(?:\.\d*)?|\.\d+)$/.test(token)) throw new Error("Finish the calculation before applying it.");
            const [whole, decimal = ""] = token.split(".");
            return fraction(BigInt(`${whole || "0"}${decimal}`), 10n ** BigInt(decimal.length));
        };
        const product = (): Fraction => {
            let left = primary();
            while (tokens[index] === "*" || tokens[index] === "/") {
                const operator = tokens[index++];
                const right = primary();
                left = operator === "*"
                    ? fraction(left.numerator * right.numerator, left.denominator * right.denominator)
                    : fraction(left.numerator * right.denominator, left.denominator * right.numerator);
            }
            return left;
        };
        const sum = (): Fraction => {
            let left = product();
            while (tokens[index] === "+" || tokens[index] === "-") {
                const operator = tokens[index++];
                const right = product();
                left = fraction(left.numerator * right.denominator +
                    (operator === "+" ? 1n : -1n) * right.numerator * left.denominator,
                left.denominator * right.denominator);
            }
            return left;
        };
        const result = sum();
        if (index !== tokens.length) throw new Error("Enter an operator between amounts.");
        if (result.numerator <= 0n) return invalid("The expense amount must be greater than zero.");
        const scaled = result.numerator * 10n ** BigInt(amountDigits);
        const remainder = scaled % result.denominator;
        const units = scaled / result.denominator + (remainder * 2n >= result.denominator ? 1n : 0n);
        if (units === 0n) return invalid("The result rounds to zero. Enter a larger amount.");
        return { amount: unitsToDecimal(units, amountDigits), error: null, rounded: remainder !== 0n };
    } catch (error) {
        return invalid(error instanceof Error ? error.message : "Enter a valid calculation.");
    }
}
