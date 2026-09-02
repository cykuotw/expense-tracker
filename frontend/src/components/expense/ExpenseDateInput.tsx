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
    ...props
}: ExpenseDateInputProps) {
    return (
        <div className={cn("expense-form-date-input", className)}>
            <Input
                type="text"
                variant="expense"
                value={formatDate(value)}
                readOnly
                disabled={disabled}
                className="expense-form-date-input__display"
                tabIndex={-1}
                aria-hidden="true"
            />
            <input
                {...props}
                className="expense-form-date-input__native"
                type="date"
                id={id}
                name={name}
                value={value}
                onChange={onChange}
                required={required}
                disabled={disabled}
            />
            <span className="expense-form-date-input__icon" aria-hidden="true">
                <Icon path={mdiCalendarMonthOutline} size={0.9} />
            </span>
        </div>
    );
}
