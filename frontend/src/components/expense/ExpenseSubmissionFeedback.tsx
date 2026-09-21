import { useEffect, useRef } from "react";

interface ExpenseSubmissionFeedbackProps {
    error: string | null;
    errorTitle: string;
    idPrefix: string;
    retryDisabled: boolean;
    saving: boolean;
    savingLabel: string;
}

export default function ExpenseSubmissionFeedback({
    error,
    errorTitle,
    idPrefix,
    retryDisabled,
    saving,
    savingLabel,
}: ExpenseSubmissionFeedbackProps) {
    const errorRef = useRef<HTMLDivElement>(null);
    const titleId = `${idPrefix}-submission-error-title`;
    const descriptionId = `${idPrefix}-submission-error-description`;

    useEffect(() => {
        if (error) {
            errorRef.current?.focus();
        }
    }, [error]);

    return (
        <>
            {error ? (
                <div
                    ref={errorRef}
                    role="alert"
                    tabIndex={-1}
                    aria-labelledby={titleId}
                    aria-describedby={descriptionId}
                    className="mb-5 rounded-2xl border border-destructive/30 bg-destructive/5 p-4 outline-none focus-visible:ring-2 focus-visible:ring-destructive/60"
                >
                    <h2 id={titleId} className="font-semibold text-destructive">
                        {errorTitle}
                    </h2>
                    <p
                        id={descriptionId}
                        className="mt-1 text-sm leading-6 text-foreground/70"
                    >
                        {error}
                    </p>
                    <button
                        type="submit"
                        className="ui-button ui-button-outline mt-3 min-h-11 px-4"
                        disabled={retryDisabled || saving}
                    >
                        Try again
                    </button>
                </div>
            ) : null}
            <p className="sr-only" role="status" aria-live="polite">
                {saving ? savingLabel : ""}
            </p>
        </>
    );
}
