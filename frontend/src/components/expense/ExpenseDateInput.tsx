import { useRef } from "react";
import type * as React from "react";
import Icon from "@mdi/react";
import { mdiCalendarMonthOutline } from "@mdi/js";

import { cn } from "../../lib/utils";
import { Input } from "../ui/input";

type ExpenseDateInputProps = Omit<
    React.ComponentProps<"input">,
    "className" | "type" | "value" | "onChange"
> & {
    className?: string;
    value: string;
    onChange: React.ChangeEventHandler<HTMLInputElement>;
};

function formatDate(value: string) {
    return /^\d{4}-\d{2}-\d{2}$/.test(value)
        ? value.replace(/-/g, "/")
        : value;
}

export function ExpenseDateInput({
    className,
    id,
    name,
    value,
    onChange,
    required,
    disabled,
    onClick,
    onKeyDown,
    ...props
}: ExpenseDateInputProps) {
    const nativeInputRef = useRef<HTMLInputElement>(null);

    const openPicker = () => {
        const input = nativeInputRef.current;
        if (!input || input.disabled) return;

        try {
            const pickerInput = input as HTMLInputElement & {
                showPicker?: () => void;
            };

            if (typeof pickerInput.showPicker === "function") {
                pickerInput.showPicker();
                return;
            }
        } catch {
            // Fall through to the browser's native click behavior.
        }

        input.click();
    };

    return (
        <div className={cn("expense-form-date-input", className)}>
            <Input
                {...props}
                id={id}
                type="text"
                variant="expense"
                value={formatDate(value)}
                readOnly
                disabled={disabled}
                aria-haspopup="dialog"
                onClick={(event) => {
                    onClick?.(event);
                    if (!event.defaultPrevented) openPicker();
                }}
                onKeyDown={(event) => {
                    onKeyDown?.(event);
                    if (
                        !event.defaultPrevented &&
                        (event.key === "Enter" || event.key === " ")
                    ) {
                        event.preventDefault();
                        openPicker();
                    }
                }}
            />
            <input
                ref={nativeInputRef}
                className="expense-form-date-input__native"
                type="date"
                name={name}
                value={value}
                onChange={onChange}
                required={required}
                disabled={disabled}
                tabIndex={-1}
                aria-hidden="true"
            />
            <button
                type="button"
                className="expense-form-date-input__button"
                aria-label="Choose expense date"
                onClick={openPicker}
                disabled={disabled}
            >
                <Icon path={mdiCalendarMonthOutline} size={0.9} />
            </button>
        </div>
    );
}
