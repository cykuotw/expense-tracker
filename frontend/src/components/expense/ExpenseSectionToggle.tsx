import { useId, type ReactNode } from "react";
import Icon from "@mdi/react";
import { mdiChevronDown } from "@mdi/js";
import { cn } from "@/lib/utils";

export function ExpenseSectionToggle({ expanded, controls, onToggle, children, summary }: {
    expanded: boolean;
    controls: string;
    onToggle: () => void;
    children: ReactNode;
    summary?: ReactNode;
}) {
    const titleId = useId();
    const summaryId = useId();
    return (
        <button type="button" aria-expanded={expanded} aria-controls={controls} onClick={onToggle}
            aria-labelledby={summary ? titleId : undefined} aria-describedby={summary ? summaryId : undefined}
            className="flex min-h-11 w-full items-center justify-between gap-3 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
            <span className="flex min-w-0 flex-col gap-1">
                <span id={titleId}>{children}</span>
                {summary ? <span id={summaryId} className="text-sm font-normal leading-relaxed text-muted-foreground">{summary}</span> : null}
            </span>
            <Icon path={mdiChevronDown} size={0.85} aria-hidden="true"
                className={cn("shrink-0", expanded && "rotate-180")} />
        </button>
    );
}
