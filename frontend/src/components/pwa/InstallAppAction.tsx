import { useEffect, useRef, useState } from "react";

import { usePWAInstall } from "../../hooks/usePWAInstall";

interface InstallAppActionProps {
    className?: string;
    listItem?: boolean;
    onOpen?: () => void;
}

export default function InstallAppAction({
    className,
    listItem = false,
    onOpen,
}: InstallAppActionProps) {
    const { canInstall, isIOS, requestInstall } = usePWAInstall();
    const [instructionsOpen, setInstructionsOpen] = useState(false);
    const closeButtonRef = useRef<HTMLButtonElement>(null);

    useEffect(() => {
        if (!instructionsOpen) return;

        closeButtonRef.current?.focus();
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === "Escape") setInstructionsOpen(false);
        };
        document.addEventListener("keydown", closeOnEscape);
        return () => document.removeEventListener("keydown", closeOnEscape);
    }, [instructionsOpen]);

    if (!canInstall) return null;

    const openInstall = () => {
        onOpen?.();
        if (isIOS) {
            setInstructionsOpen(true);
            return;
        }
        void requestInstall();
    };

    const action = (
        <button type="button" className={className} onClick={openInstall}>
            Install app
        </button>
    );
    const instructions = instructionsOpen ? (
                <div
                    className="fixed inset-0 z-[100] flex items-end justify-center bg-neutral/50 p-0 backdrop-blur-sm sm:items-center sm:p-6"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) {
                            setInstructionsOpen(false);
                        }
                    }}
                >
                    <section
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby="install-app-title"
                        className="w-full rounded-t-[2rem] border border-base-300 bg-base-100 p-6 shadow-2xl sm:max-w-md sm:rounded-[2rem] sm:p-7"
                    >
                        <div className="section-label">Install the app</div>
                        <h2
                            id="install-app-title"
                            className="mt-2 text-xl font-semibold tracking-[-0.02em] text-base-content"
                        >
                            Add Expense Tracker to your Home Screen
                        </h2>
                        <ol className="mt-4 list-decimal space-y-2 pl-5 text-sm leading-6 text-base-content/70">
                            <li>Tap the Share button in Safari.</li>
                            <li>Choose Add to Home Screen.</li>
                            <li>Tap Add to finish.</li>
                        </ol>
                        <div className="mt-7 flex justify-end">
                            <button
                                ref={closeButtonRef}
                                type="button"
                                className="btn btn-neutral min-h-11"
                                onClick={() => setInstructionsOpen(false)}
                            >
                                Got it
                            </button>
                        </div>
                    </section>
                </div>
            ) : null;

    return listItem ? (
        <li className="w-full">
            {action}
            {instructions}
        </li>
    ) : (
        <>
            {action}
            {instructions}
        </>
    );
}
