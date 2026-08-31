import { Link } from "react-router-dom";
import { GroupCardData } from "../../types/group";
import Icon from "@mdi/react";
import { getGroupTypePresentation } from "../../lib/groupTypePresentation";

export default function GroupCard(groupData: GroupCardData) {
    const hasDescription = Boolean(groupData.description?.trim());
    const type = getGroupTypePresentation(groupData.groupType);
    const balanceLabel =
        groupData.balanceStatus === "settled"
            ? "Settled"
            : groupData.balanceStatus === "owed"
              ? `You are owed $${groupData.balanceAmount} ${groupData.currency}`
              : `You owe $${groupData.balanceAmount} ${groupData.currency}`;

    const balanceClass =
        groupData.balanceStatus === "settled"
            ? "bg-muted text-foreground/70"
            : groupData.balanceStatus === "owed"
              ? "bg-success/12 text-success"
              : "bg-destructive/12 text-destructive";

    return (
        <div className="group h-full w-full">
            <Link to={`/group/${groupData.id}`} className="block h-full">
                <div className="panel-card flex h-full w-full flex-col rounded-[1.75rem] p-6 transition duration-300 hover:-translate-y-1 hover:shadow-lg">
                    <div className="flex items-start justify-between gap-3">
                        <div className="flex min-w-0 items-center gap-3">
                            <span className={`flex size-10 shrink-0 items-center justify-center rounded-xl ${type.iconClassName}`}><Icon path={type.icon} size={1} /></span>
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
                </div>
            </Link>
        </div>
    );
}
