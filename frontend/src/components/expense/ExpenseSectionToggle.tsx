import type { ReactNode } from "react";
import Icon from "@mdi/react";
import { mdiChevronDown } from "@mdi/js";
import { cn } from "@/lib/utils";

export function ExpenseSectionToggle({ expanded, controls, onToggle, children }: {
    expanded: boolean;
    controls: string;
    onToggle: () => void;
    children: ReactNode;
}) {
    return (
        <button type="button" aria-expanded={expanded} aria-controls={controls} onClick={onToggle}
            className="flex min-h-11 w-full items-center justify-between gap-3 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
            <span>{children}</span>
            <Icon path={mdiChevronDown} size={0.85} aria-hidden="true"
                className={cn("shrink-0", expanded && "rotate-180")} />
        </button>
    );
}
