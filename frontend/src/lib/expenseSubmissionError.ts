import { getResponseError } from "./api";

export type ExpenseSubmissionKind = "create" | "update";

const FALLBACK_MESSAGES: Record<ExpenseSubmissionKind, string> = {
    create: "We couldn't save this expense. Check your connection and try again.",
    update: "We couldn't save your changes. Check your connection and try again.",
};

const CODE_MESSAGES: Record<string, string> = {
    invalid_occurred_on: "Choose a valid expense date and try again.",
    currency_mismatch:
        "This group's currency settings changed. Reload the page before trying again.",
    invalid_money: "Check the expense amount and split, then try again.",
    invalid_action: "Check the expense details and split, then try again.",
    idempotency_key_conflict:
        "We couldn't safely retry this expense. Reload the page and review it before submitting again.",
    balance_ledger_conflict:
        "This group changed while you were saving. Reload the page and try again.",
    permission_denied:
        "You no longer have permission to save expenses in this group.",
    user_not_permitted:
        "You no longer have permission to save expenses in this group.",
    forbidden: "You no longer have permission to save expenses in this group.",
    expense_not_exist:
        "This expense is no longer available. Return to the group and refresh the list.",
};

export function getExpenseSubmissionFallback(kind: ExpenseSubmissionKind) {
    return FALLBACK_MESSAGES[kind];
}

export async function getExpenseSubmissionErrorMessage(
    response: Response,
    kind: ExpenseSubmissionKind,
) {
    const fallback = getExpenseSubmissionFallback(kind);
    const { code } = await getResponseError(response, fallback);

    return code ? CODE_MESSAGES[code] ?? fallback : fallback;
}
