import { Link } from "react-router-dom";
import { GroupCardData } from "../../types/group";

export default function GroupCard(groupData: GroupCardData) {
    const hasDescription = Boolean(groupData.description?.trim());
    const balanceLabel =
        groupData.balanceStatus === "settled"
            ? "Settled"
            : groupData.balanceStatus === "owed"
              ? `You are owed $${groupData.balanceAmount} ${groupData.currency}`
              : `You owe $${groupData.balanceAmount} ${groupData.currency}`;

    const balanceClass =
        groupData.balanceStatus === "settled"
            ? "bg-base-200 text-base-content/70"
            : groupData.balanceStatus === "owed"
              ? "bg-success/12 text-success"
              : "bg-error/12 text-error";

    return (
        <div className="group h-full w-full">
            <Link to={`/group/${groupData.id}`} className="block h-full">
                <div className="panel-card flex h-full w-full flex-col rounded-[1.75rem] p-6 transition duration-300 hover:-translate-y-1 hover:shadow-lg">
                    <div className="flex items-start justify-between gap-3">
                        <div className="text-lg font-semibold tracking-[-0.02em]">
                            {groupData.groupName}
                        </div>
                        <div className="rounded-full bg-primary/10 px-3 py-1 text-xs uppercase tracking-wider text-primary">
                            Group
                        </div>
                    </div>
                    <p
                        className={`mt-4 text-sm leading-6 break-words ${
                            hasDescription
                                ? "text-base-content/70"
                                : "italic text-base-content/45"
                        }`}
                    >
                        {hasDescription
                            ? groupData.description
                            : "No description yet."}
                    </p>
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
