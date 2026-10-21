import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import OCRImprovementSettings from "./OCRImprovementSettings";

const { api } = vi.hoisted(() => ({ api: vi.fn() }));
vi.mock("@/lib/api", () => ({ apiFetch: api, getResponseErrorMessage: async () => "Preference unavailable" }));
const preference = { enabled: false, policyVersion: "ocr-improvement-v1", revision: 0, purgePending: false, captureAvailable: true };
const response = (value: unknown, status = 200) => ({ ok: status < 400, status, json: async () => value });
afterEach(() => { cleanup(); api.mockReset(); });

describe("OCRImprovementSettings", () => {
    it("allows opting out after OCR access is revoked while preventing a new opt-in", async () => {
        api.mockResolvedValueOnce(response({ ...preference, enabled: true, revision: 2 })).mockResolvedValueOnce(response({ ...preference, revision: 3 }));
        render(<OCRImprovementSettings granted={false} />);
        const toggle = await screen.findByRole("switch");
        expect(toggle).not.toBeDisabled();
        fireEvent.click(toggle);
        expect(toggle).toBeDisabled();
        fireEvent.click(screen.getByRole("button", { name: "Save sharing preference" }));
        await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
        expect(JSON.parse(api.mock.calls[1][1].body)).toEqual({ enabled: false, policyVersion: "ocr-improvement-v1", expectedRevision: 2 });
    });
    it("requires explicit saving and reloads a stale revision after conflict", async () => {
        api.mockResolvedValueOnce(response(preference)).mockResolvedValueOnce(response({}, 409)).mockResolvedValueOnce(response({ ...preference, enabled: true, revision: 3 }));
        render(<OCRImprovementSettings granted />);
        const toggle = await screen.findByRole("switch", { name: "Share receipt samples for improvement" });
        fireEvent.click(toggle);
        expect(api).toHaveBeenCalledTimes(1);
        fireEvent.click(screen.getByRole("button", { name: "Save sharing preference" }));
        expect(await screen.findByRole("alert")).toHaveTextContent("Review the current setting");
        expect(api).toHaveBeenCalledTimes(3);
        expect(toggle).toHaveAttribute("aria-checked", "true");
        expect(screen.getByRole("button", { name: "Save sharing preference" })).toBeDisabled();
    });

    it("shows pending deletion and allows checking it without enabling sharing", async () => {
        api.mockResolvedValueOnce(response({ ...preference, purgePending: true, revision: 4 })).mockResolvedValueOnce(response({ ...preference, revision: 4 }));
        render(<OCRImprovementSettings granted />);
        expect(await screen.findByRole("switch")).toBeDisabled();
        fireEvent.click(screen.getByRole("button", { name: "Check deletion" }));
        await waitFor(() => expect(screen.queryByRole("button", { name: "Check deletion" })).toBeNull());
        expect(api).toHaveBeenLastCalledWith("/account/ocr-improvement/purge", expect.objectContaining({ method: "POST" }));
    });
});
