import { Link } from "react-router-dom";
import { useRef } from "react";
import { GroupCardData } from "../../types/group";
import Icon from "@mdi/react";
import { mdiArchiveOutline, mdiClose } from "@mdi/js";
import { useNavigationState } from "../../hooks/navigation";
import { Button } from "../ui/button";
import { getGroupTypePresentation } from "../../lib/groupTypePresentation";

export default function GroupCard(groupData: GroupCardData) {
    const [suggestionDismissed, setSuggestionDismissed] = useNavigationState<boolean>(`home.archive.dismissed.${groupData.id}`, false);
    const groupLink = useRef<HTMLAnchorElement>(null);
    const hasDescription = Boolean(groupData.description?.trim());
    const type = getGroupTypePresentation(groupData.groupType);
    const estimatePrefix = groupData.usesSettlementPreview ? "Estimated · " : "";
    const approximatePrefix = groupData.usesSettlementPreview ? "≈ " : "";
    const balanceSubject = groupData.usesSettlementPreview ? "you" : "You";
    const balanceLabel =
        groupData.balanceStatus === "settled"
            ? groupData.usesSettlementPreview ? "Estimated net balance: 0" : "Your net balance: 0"
            : groupData.balanceStatus === "preview_unavailable"
              ? "Settlement preview unavailable"
            : groupData.balanceStatus === "owed"
              ? `${estimatePrefix}${balanceSubject} are owed ${approximatePrefix}${groupData.balanceAmount} ${groupData.currency}`
              : `${estimatePrefix}${balanceSubject} owe ${approximatePrefix}${groupData.balanceAmount} ${groupData.currency}`;

    const balanceClass =
        groupData.balanceStatus === "settled"
            ? "bg-muted text-foreground/70"
            : groupData.balanceStatus === "preview_unavailable"
              ? "bg-muted text-foreground/70"
            : groupData.balanceStatus === "owed"
              ? "bg-success/12 text-success"
              : "bg-destructive/12 text-destructive";

    return (
        <div className="panel-card group flex h-full w-full flex-col rounded-[1.75rem] p-6 transition duration-300 hover:-translate-y-1 hover:shadow-lg">
            <Link ref={groupLink} to={`/group/${groupData.id}`} className="flex flex-1 flex-col rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                    <div className="flex items-start justify-between gap-3">
                        <div className="flex min-w-0 items-center gap-3">
                            <span className={`flex size-10 shrink-0 items-center justify-center rounded-xl ${type.iconClassName}`} aria-hidden="true"><Icon path={type.icon} size={1} /></span>
                            <div className="text-lg font-semibold tracking-[-0.02em]">{groupData.groupName}</div>
                        </div>
                        <div className={`rounded-full px-3 py-1 text-xs uppercase tracking-wider ${type.iconClassName}`}>{type.label}</div>
                    </div>
                    {hasDescription && <p className="mt-4 break-words text-sm leading-6 text-foreground/70">{groupData.description}</p>}
                    <div className="mt-auto pt-6">
                        <div
                            className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${balanceClass}`}
                        >
                            {balanceLabel}
                        </div>
                    </div>
                    <div className="pt-4 text-xs font-semibold uppercase tracking-[0.2em] text-primary">
                        Open
                    </div>
            </Link>
            {groupData.archiveSuggested && !suggestionDismissed ? (
                <aside className="mt-3 flex items-center gap-2 border-t border-border pt-2" aria-label={`Archive suggestion for ${groupData.groupName}`}>
                    <p className="min-w-0 flex-1 text-xs leading-5 text-muted-foreground">Inactive for 90+ days</p>
                    <Button asChild variant="muted" size="sm" className="min-h-11"><Link to={`/group/${groupData.id}/edit#status`}><Icon path={mdiArchiveOutline} data-icon="inline-start" aria-hidden="true" />Archive</Link></Button>
                    <Button variant="ghost" size="icon" className="size-11" aria-label={`Dismiss archive suggestion for ${groupData.groupName}`} title="Not now" onClick={() => { setSuggestionDismissed(true); groupLink.current?.focus(); }}><Icon path={mdiClose} aria-hidden="true" /></Button>
                </aside>
            ) : null}
        </div>
    );
}
