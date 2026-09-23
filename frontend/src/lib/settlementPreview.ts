import { Balance, BalanceData } from "../types/balance";

interface ScaledDecimal {
    units: bigint;
    scale: number;
}

export interface EstimatedSettlementEntry {
    counterpartyUserId: string;
    label: string;
    amount: string;
    currency: string;
    tone: string;
}

const SIGNED_DECIMAL = /^(-?)(\d+)(?:\.(\d+))?$/;

function parseDecimal(value: string): ScaledDecimal | null {
    const match = SIGNED_DECIMAL.exec(value);
    if (!match) return null;
    const fraction = match[3] ?? "";
    const magnitude = BigInt(`${match[2]}${fraction}`);
    return {
        units: match[1] === "-" ? -magnitude : magnitude,
        scale: fraction.length,
    };
}

function addDecimals(left: ScaledDecimal, right: ScaledDecimal): ScaledDecimal {
    const scale = Math.max(left.scale, right.scale);
    return {
        units:
            left.units * 10n ** BigInt(scale - left.scale) +
            right.units * 10n ** BigInt(scale - right.scale),
        scale,
    };
}

function formatDecimal(value: ScaledDecimal): string {
    const negative = value.units < 0n;
    const absolute = negative ? -value.units : value.units;
    if (value.scale === 0) return `${negative ? "-" : ""}${absolute}`;
    const padded = absolute.toString().padStart(value.scale + 1, "0");
    const integer = padded.slice(0, -value.scale);
    const fraction = padded.slice(-value.scale).replace(/0+$/, "");
    return `${negative ? "-" : ""}${integer}${fraction ? `.${fraction}` : ""}`;
}

function counterpartyFor(balance: Balance, currentUser: string) {
    if (balance.receiverUserId === currentUser) {
        return { id: balance.senderUserId, name: balance.senderUsername };
    }
    if (balance.senderUserId === currentUser) {
        return { id: balance.receiverUserId, name: balance.receiverUsername };
    }
    return null;
}

export function buildEstimatedSettlementEntries(
    balanceData: BalanceData
): EstimatedSettlementEntry[] {
    const preview = balanceData.settlementPreview;
    if (!preview?.complete) return [];

    const balancesById = new Map(
        balanceData.balances.map((balance) => [balance.id, balance])
    );
    const totals = new Map<string, {
        name: string;
        value: ScaledDecimal;
    }>();

    for (const contribution of preview.contributions) {
        if (contribution.previewAmount === null) continue;
        const balance = balancesById.get(contribution.balanceId);
        if (!balance) continue;
        const counterparty = counterpartyFor(balance, balanceData.currentUser);
        const amount = parseDecimal(contribution.previewAmount);
        if (!counterparty || !amount) continue;

        const current = totals.get(counterparty.id);
        totals.set(counterparty.id, {
            name: counterparty.name,
            value: current ? addDecimals(current.value, amount) : amount,
        });
    }

    return Array.from(totals, ([counterpartyUserId, total]) => {
        const amount = formatDecimal(total.value);
        const isOwed = total.value.units > 0n;
        const owes = total.value.units < 0n;
        return {
            counterpartyUserId,
            label: isOwed
                ? `${total.name} owes you`
                : owes
                    ? `You owe ${total.name}`
                    : `Approximately settled with ${total.name}`,
            amount: owes ? amount.slice(1) : amount,
            currency: preview.currency,
            tone: isOwed
                ? "text-success"
                : owes
                    ? "text-destructive"
                    : "text-foreground/65",
        };
    });
}
