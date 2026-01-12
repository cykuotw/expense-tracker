import { Link } from "react-router-dom";
import { GroupCardData } from "../../types/group";

export default function GroupCard(groupData: GroupCardData) {
    return (
        <div className="group h-full w-full">
            <Link to={`/group/${groupData.id}`} className="block h-full">
                <div className="flex h-full w-full flex-col rounded-2xl border border-base-300 bg-base-100/90 p-5 shadow-sm transition duration-300 hover:-translate-y-1 hover:shadow-lg">
                    <div className="flex items-start justify-between gap-3">
                        <div className="text-lg font-semibold">
                            {groupData.groupName}
                        </div>
                        <div className="rounded-full bg-base-200 px-3 py-1 text-xs uppercase tracking-wider text-base-content/70">
                            Group
                        </div>
                    </div>
                    <p className="mt-3 text-sm text-base-content/70 break-words">
                        {groupData.description || "No description yet."}
                    </p>
                    <div className="mt-auto pt-4 text-xs font-semibold uppercase tracking-[0.2em] text-neutral">
                        Open
                    </div>
                </div>
            </Link>
        </div>
    );
}
