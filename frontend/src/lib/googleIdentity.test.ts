import { afterEach, describe, expect, it, vi } from "vitest";

describe("loadGoogleAccountsId", () => {
    afterEach(() => {
        document.head.innerHTML = "";
        document.body.innerHTML = "";
        delete window.google;
        vi.resetModules();
        vi.restoreAllMocks();
    });

    it("resolves immediately when GIS is already available", async () => {
        const { loadGoogleAccountsId } = await import("./googleIdentity");
        const googleAccountsId = {
            initialize: vi.fn(),
            renderButton: vi.fn(),
        };

        window.google = {
            accounts: {
                id: googleAccountsId,
            },
        };

        await expect(loadGoogleAccountsId()).resolves.toBe(googleAccountsId);
        expect(document.getElementById("google-identity-services-client")).toBeNull();
    });

    it("loads the GIS script and resolves after the load event", async () => {
        const { loadGoogleAccountsId } = await import("./googleIdentity");
        const googleAccountsId = {
            initialize: vi.fn(),
            renderButton: vi.fn(),
        };

        const loadPromise = loadGoogleAccountsId();
        const script = document.getElementById(
            "google-identity-services-client"
        ) as HTMLScriptElement | null;

        expect(script).not.toBeNull();
        expect(script?.src).toContain("https://accounts.google.com/gsi/client");

        window.google = {
            accounts: {
                id: googleAccountsId,
            },
        };
        script?.dispatchEvent(new Event("load"));

        await expect(loadPromise).resolves.toBe(googleAccountsId);
    });

    it("rejects when the GIS script fails to load", async () => {
        const { loadGoogleAccountsId } = await import("./googleIdentity");
        const loadPromise = loadGoogleAccountsId();
        const script = document.getElementById(
            "google-identity-services-client"
        ) as HTMLScriptElement | null;

        script?.dispatchEvent(new Event("error"));

        await expect(loadPromise).rejects.toThrow(
            "Failed to load the Google sign-in SDK."
        );
    });
});
