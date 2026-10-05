import Icon from "@mdi/react";
import { mdiArchiveOutline } from "@mdi/js";
import { Badge } from "../ui/badge";

export default function ArchivedStatus({ label = "Archived" }: { label?: string }) {
    return (
        <Badge variant="outline" className="min-h-7 gap-2 px-3">
            <Icon path={mdiArchiveOutline} aria-hidden="true" />
            {label}
        </Badge>
    );
}
