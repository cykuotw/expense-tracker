import { afterEach, describe, expect, it, vi } from "vitest";

import {
    asArray,
    getResponseError,
    getResponseErrorMessage,
    submitFrontendRenderError,
} from "./api";

const fallback = "Fallback message";

afterEach(() => {
    vi.unstubAllGlobals();
});

function jsonResponse(body: unknown, status = 400) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

describe("getResponseErrorMessage", () => {
    it("prefers the backend error field", async () => {
        const response = jsonResponse({
            error: "user not permitted",
            message: "legacy message",
        });

        await expect(
            getResponseErrorMessage(response, fallback)
        ).resolves.toBe("user not permitted");
    });

    it("supports the legacy message field", async () => {
        const response = jsonResponse({ message: "legacy message" });

        await expect(
            getResponseErrorMessage(response, fallback)
        ).resolves.toBe("legacy message");
    });

    it("preserves non-empty plain-text errors", async () => {
        const response = new Response("  Service unavailable  ", {
            status: 503,
            headers: { "Content-Type": "text/plain" },
        });

        await expect(
            getResponseErrorMessage(response, fallback)
        ).resolves.toBe("Service unavailable");
    });

    it.each(["", "   \n\t"])(
        "uses the fallback for an empty body %#",
        async (body) => {
            const response = new Response(body, {
                status: 500,
                headers: { "Content-Type": "text/plain" },
            });

            await expect(
                getResponseErrorMessage(response, fallback)
            ).resolves.toBe(fallback);
        }
    );

    it("uses the fallback for malformed JSON", async () => {
        const response = new Response("{invalid", {
            status: 400,
            headers: { "Content-Type": "application/json" },
        });

        await expect(
            getResponseErrorMessage(response, fallback)
        ).resolves.toBe(fallback);
    });

    it.each([
        {},
        { error: "", message: "   " },
        { error: 123 },
        null,
        ["unsupported"],
    ])("uses the fallback for unsupported JSON %#", async (body) => {
        const response = jsonResponse(body);

        await expect(
            getResponseErrorMessage(response, fallback)
        ).resolves.toBe(fallback);
    });
});

describe("getResponseError", () => {
    it("preserves stable backend error codes", async () => {
        const response = jsonResponse({
            error: "invitation required",
            code: "INVITATION_REQUIRED",
        });

        await expect(getResponseError(response, fallback)).resolves.toEqual({
            message: "invitation required",
            code: "INVITATION_REQUIRED",
        });
    });
});

describe("asArray", () => {
    it("keeps valid arrays and normalizes non-array API payloads", () => {
        expect(asArray<string>(["member"])).toEqual(["member"]);
        expect(asArray<string>(null)).toEqual([]);
        expect(asArray<string>({ members: [] })).toEqual([]);
    });
});

describe("submitFrontendRenderError", () => {
    it("sends only the occurrence ID through the CSRF-protected client", async () => {
        const fetchMock = vi
            .fn()
            .mockResolvedValueOnce(
                jsonResponse({ csrfToken: "csrf-token" }, 200),
            )
            .mockResolvedValueOnce(new Response(null, { status: 202 }));
        vi.stubGlobal("fetch", fetchMock);

        await submitFrontendRenderError(
            "4f64807d-f824-4d3b-9cb9-2cd206548639",
        );

        expect(fetchMock).toHaveBeenCalledTimes(2);
        const [, request] = fetchMock.mock.calls[1];
        expect(request).toMatchObject({
            method: "POST",
            credentials: "include",
            body: '{"occurrenceId":"4f64807d-f824-4d3b-9cb9-2cd206548639"}',
        });
        expect(new Headers(request.headers).get("X-CSRF-Token")).toBe(
            "csrf-token",
        );
    });
});
