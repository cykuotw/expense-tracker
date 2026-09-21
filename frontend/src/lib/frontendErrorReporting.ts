import { submitFrontendRenderError } from "./api";

const reportStorageKey = "expense-tracker:frontend-render-error:v1";

type ReporterDependencies = {
    enabled: boolean;
    randomUUID: () => string;
    storage: Pick<Storage, "getItem" | "setItem"> | null;
    submit: (occurrenceId: string) => Promise<void>;
};

export function createFrontendErrorReporter({
    enabled,
    randomUUID,
    storage,
    submit,
}: ReporterDependencies): () => void {
    let attempted = false;

    return () => {
        if (!enabled || attempted) {
            return;
        }
        attempted = true;

        try {
            if (storage?.getItem(reportStorageKey) === "reported") {
                return;
            }
            storage?.setItem(reportStorageKey, "reported");
        } catch {
            // Storage may be unavailable; the in-memory guard still caps attempts.
        }

        try {
            const occurrenceId = randomUUID();
            void submit(occurrenceId).catch(() => undefined);
        } catch {
            // Telemetry must never interfere with the recovery screen.
        }
    };
}

function getSessionStorage(): Storage | null {
    try {
        return globalThis.sessionStorage;
    } catch {
        return null;
    }
}

export const reportFrontendRenderError = createFrontendErrorReporter({
    enabled: import.meta.env.PROD,
    randomUUID: () => crypto.randomUUID(),
    storage: getSessionStorage(),
    submit: submitFrontendRenderError,
});
