import { useRef } from "react";

import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

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
    const cancelButtonRef = useRef<HTMLButtonElement>(null);

    return (
        <AlertDialog
            open
            onOpenChange={(open) => {
                if (!open && !busy) onCancel();
            }}
        >
            <AlertDialogContent
                onEscapeKeyDown={(event) => {
                    if (busy) event.preventDefault();
                }}
                onOpenAutoFocus={(event) => {
                    event.preventDefault();
                    cancelButtonRef.current?.focus();
                }}
            >
                <AlertDialogHeader>
                    <div className="section-label">Please confirm</div>
                    <AlertDialogTitle>{title}</AlertDialogTitle>
                    <AlertDialogDescription>
                        {description}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                {detail ? (
                    <div className="rounded-2xl border border-border bg-muted/60 px-4 py-3 text-sm font-medium text-foreground">
                        {detail}
                    </div>
                ) : null}
                <AlertDialogFooter>
                    <Button
                        ref={cancelButtonRef}
                        type="button"
                        variant="ghost"
                        disabled={busy}
                        onClick={onCancel}
                    >
                        Cancel
                    </Button>
                    <Button
                        type="button"
                        variant={tone === "danger" ? "destructive" : "default"}
                        disabled={busy}
                        onClick={onConfirm}
                    >
                        {busy ? (
                            <Spinner aria-hidden="true" data-icon="inline-start" />
                        ) : null}
                        {busy ? "Updating…" : confirmLabel}
                    </Button>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
