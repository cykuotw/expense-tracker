import { Rule } from "../types/splitRule";

export function orientTwoPersonRuleForViewer(
    rule: Rule,
    payerUserId: string,
    currentUserId: string
): Rule {
    if (!payerUserId || !currentUserId) return rule;

    if (rule === Rule.YouHalf || rule === Rule.OtherHalf) {
        return payerUserId === currentUserId ? Rule.YouHalf : Rule.OtherHalf;
    }

    if (rule === Rule.YouFull || rule === Rule.OtherFull) {
        return payerUserId === currentUserId ? Rule.YouFull : Rule.OtherFull;
    }

    return rule;
}
