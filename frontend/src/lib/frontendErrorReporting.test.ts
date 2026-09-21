import { describe, expect, it, vi } from "vitest";

import { createFrontendErrorReporter } from "./frontendErrorReporting";

describe("createFrontendErrorReporter", () => {
    it("submits one generated occurrence without accepting error details", async () => {
        const submit = vi.fn().mockResolvedValue(undefined);
        const storage = {
            getItem: vi.fn().mockReturnValue(null),
            setItem: vi.fn(),
        };
        const reporter = createFrontendErrorReporter({
            enabled: true,
            randomUUID: () => "4f64807d-f824-4d3b-9cb9-2cd206548639",
            storage,
            submit,
        });

        reporter();
        reporter();

        expect(submit).toHaveBeenCalledOnce();
        expect(submit).toHaveBeenCalledWith(
            "4f64807d-f824-4d3b-9cb9-2cd206548639",
        );
        expect(storage.setItem).toHaveBeenCalledWith(
            "expense-tracker:frontend-render-error:v1",
            "reported",
        );
    });

    it("does not report when disabled or previously sampled", () => {
        const submit = vi.fn().mockResolvedValue(undefined);
        const disabled = createFrontendErrorReporter({
            enabled: false,
            randomUUID: crypto.randomUUID,
            storage: null,
            submit,
        });
        const sampled = createFrontendErrorReporter({
            enabled: true,
            randomUUID: crypto.randomUUID,
            storage: { getItem: () => "reported", setItem: vi.fn() },
            submit,
        });

        disabled();
        sampled();

        expect(submit).not.toHaveBeenCalled();
    });

    it("silently contains storage, UUID, and request failures", () => {
        const storageFailure = createFrontendErrorReporter({
            enabled: true,
            randomUUID: () => "4f64807d-f824-4d3b-9cb9-2cd206548639",
            storage: {
                getItem: () => {
                    throw new Error("storage unavailable");
                },
                setItem: vi.fn(),
            },
            submit: vi.fn().mockRejectedValue(new Error("network unavailable")),
        });
        const uuidFailure = createFrontendErrorReporter({
            enabled: true,
            randomUUID: () => {
                throw new Error("UUID unavailable");
            },
            storage: null,
            submit: vi.fn(),
        });

        expect(storageFailure).not.toThrow();
        expect(uuidFailure).not.toThrow();
    });
});
