import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";

interface ExpenseSubmitButtonProps {
    className: string;
    disabled: boolean;
    form?: string;
    iconOnly?: boolean;
    idleLabel: string;
    saving: boolean;
    savingLabel: string;
}

export default function ExpenseSubmitButton({
    className,
    disabled,
    form,
    iconOnly = false,
    idleLabel,
    saving,
    savingLabel,
}: ExpenseSubmitButtonProps) {
    const label = saving ? savingLabel : idleLabel;

    return (
        <button
            type="submit"
            form={form}
            className={className}
            aria-label={label}
            aria-busy={saving}
            disabled={disabled || saving}
        >
            {saving ? (
                <span className="ui-spinner ui-spinner-sm" aria-hidden="true" />
            ) : (
                <Icon path={mdiCheckBold} size={1} aria-hidden="true" />
            )}
            {iconOnly ? null : label}
        </button>
    );
}
