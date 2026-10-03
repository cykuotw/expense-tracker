import Icon from "@mdi/react";
import { mdiChevronDown } from "@mdi/js";
import { Button } from "../ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from "../ui/dropdown-menu";
import { cn } from "../../lib/utils";

export interface ExpenseOrderOption<T extends string> {
    value: T;
    label: string;
}

interface ExpenseOrderPickerProps<T extends string> {
    value: T;
    options: readonly ExpenseOrderOption<T>[];
    onChange: (value: T) => void;
    label: string;
    accessibleLabel: string;
    hideLabelOnMobile?: boolean;
}

export function ExpenseOrderPicker<T extends string>({ value, options, onChange, label, accessibleLabel, hideLabelOnMobile = false }: ExpenseOrderPickerProps<T>) {
    const selectedLabel = options.find((option) => option.value === value)?.label ?? value;
    return (
        <div className="flex min-w-0 items-center justify-end gap-2">
            <span className={cn("shrink-0 text-sm text-muted-foreground", hideLabelOnMobile && "sr-only sm:not-sr-only")}>
                {label}
            </span>
            <DropdownMenu modal={false}>
                <DropdownMenuTrigger asChild>
                    <Button type="button" variant="outline" className="min-h-11 min-w-0 max-w-full rounded-2xl px-3" aria-label={`${accessibleLabel}: ${selectedLabel}`}>
                        <span className="truncate font-sans text-sm font-normal">{selectedLabel}</span>
                        <Icon path={mdiChevronDown} data-icon="inline-end" aria-hidden="true" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" sideOffset={8} className="min-w-56 rounded-2xl p-1.5 shadow-xl" aria-label={accessibleLabel}>
                    <DropdownMenuRadioGroup value={value} onValueChange={(nextValue) => {
                        const selected = options.find((option) => option.value === nextValue);
                        if (selected) onChange(selected.value);
                    }}>
                        {options.map((option) => (
                            <DropdownMenuRadioItem key={option.value} value={option.value} className="min-h-11 rounded-xl pl-3 pr-9 focus:bg-secondary focus:text-primary data-[highlighted]:bg-secondary data-[highlighted]:text-primary data-[state=checked]:bg-primary/10 data-[state=checked]:text-primary">
                                <span className="font-sans text-sm font-normal">{option.label}</span>
                            </DropdownMenuRadioItem>
                        ))}
                    </DropdownMenuRadioGroup>
                </DropdownMenuContent>
            </DropdownMenu>
        </div>
    );
}
