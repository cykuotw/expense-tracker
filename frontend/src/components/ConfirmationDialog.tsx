import { useEffect, useRef } from "react";

interface ConfirmationDialogProps {
    title: string;
    description: string;
    detail?: string;
    confirmLabel: string;
    busy?: boolean;
    tone?: "default" | "danger";
    onCancel: () => void;
    onConfirm: () => void;
}

export default function ConfirmationDialog({
    title,
    description,
    detail,
    confirmLabel,
    busy = false,
    tone = "default",
    onCancel,
    onConfirm,
}: ConfirmationDialogProps) {
    const dialogRef = useRef<HTMLDivElement>(null);
    const cancelButtonRef = useRef<HTMLButtonElement>(null);
    const busyRef = useRef(busy);
    const onCancelRef = useRef(onCancel);
    busyRef.current = busy;
    onCancelRef.current = onCancel;

    useEffect(() => {
        const previousFocus = document.activeElement as HTMLElement | null;
        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
        cancelButtonRef.current?.focus();

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape" && !busyRef.current) {
                event.preventDefault();
                onCancelRef.current();
                return;
            }
            if (event.key !== "Tab" || !dialogRef.current) return;

            const controls = Array.from(
                dialogRef.current.querySelectorAll<HTMLElement>(
                    'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
                ),
            );
            if (controls.length === 0) return;
            const first = controls[0];
            const last = controls[controls.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };

        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("keydown", handleKeyDown);
            document.body.style.overflow = previousOverflow;
            previousFocus?.focus();
        };
    }, []);

    return (
        <div
            className="fixed inset-0 z-[100] flex items-end justify-center bg-neutral/50 p-0 backdrop-blur-sm sm:items-center sm:p-6"
            onMouseDown={(event) => {
                if (event.target === event.currentTarget && !busy) onCancel();
            }}
        >
            <div
                ref={dialogRef}
                role="alertdialog"
                aria-modal="true"
                aria-labelledby="confirmation-dialog-title"
                aria-describedby="confirmation-dialog-description"
                className="w-full rounded-t-[2rem] border border-base-300 bg-base-100 p-6 shadow-2xl sm:max-w-md sm:rounded-[2rem] sm:p-7"
            >
                <div className="section-label">Please confirm</div>
                <h2
                    id="confirmation-dialog-title"
                    className="mt-2 text-xl font-semibold tracking-[-0.02em] text-base-content"
                >
                    {title}
                </h2>
                <p
                    id="confirmation-dialog-description"
                    className="mt-3 text-sm leading-6 text-base-content/70"
                >
                    {description}
                </p>
                {detail ? (
                    <div className="mt-5 rounded-2xl border border-base-300 bg-base-200/60 px-4 py-3 text-sm font-medium text-base-content">
                        {detail}
                    </div>
                ) : null}
                <div className="mt-7 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
                    <button
                        ref={cancelButtonRef}
                        type="button"
                        className="btn btn-ghost min-h-11 sm:min-w-28"
                        disabled={busy}
                        onClick={onCancel}
                    >
                        Cancel
                    </button>
                    <button
                        type="button"
                        className={`btn min-h-11 sm:min-w-32 ${
                            tone === "danger" ? "btn-error" : "btn-neutral"
                        }`}
                        disabled={busy}
                        onClick={onConfirm}
                    >
                        {busy ? (
                            <span className="loading loading-spinner loading-sm" />
                        ) : null}
                        {busy ? "Updating…" : confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    );
}
