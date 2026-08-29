import { ReactNode } from "react";

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface DropdownProps {
    label: string | ReactNode;
    side?: "top" | "bottom";
    contentClassName?: string;
    groupClassName?: string;
    children: ReactNode;
}

const Dropdown = ({
    label,
    side = "bottom",
    contentClassName,
    groupClassName,
    children,
}: DropdownProps) => (
    <DropdownMenu>
        <DropdownMenuTrigger asChild>
            <button
                type="button"
                className="flex w-full items-center justify-between gap-2 rounded-2xl px-3 py-2 text-left text-current hover:bg-muted/70"
            >
                {label}
                <svg
                    aria-hidden="true"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    className="size-4"
                >
                    <path d="m6 9 6 6 6-6" />
                </svg>
            </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
            side={side}
            className={cn("w-48 max-w-64", contentClassName)}
        >
            <DropdownMenuGroup className={groupClassName}>
                {children}
            </DropdownMenuGroup>
        </DropdownMenuContent>
    </DropdownMenu>
);

export default Dropdown;
