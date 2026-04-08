import { afterEach, describe, expect, it, vi } from "vitest";

describe("config", () => {
    afterEach(() => {
        vi.resetModules();
        vi.unstubAllEnvs();
    });

    it("allows Google OAuth to stay disabled without a client id", async () => {
        vi.stubEnv("VITE_API_ORIGIN", "http://localhost:8000");
        vi.stubEnv("VITE_API_PATH", "/api/v0");
        vi.stubEnv("VITE_GOOGLE_OAUTH_ENABLED", "false");
        vi.stubEnv("VITE_GOOGLE_CLIENT_ID", "");

        const { GOOGLE_OAUTH_ENABLED, GOOGLE_CLIENT_ID } = await import(
            "./config"
        );

        expect(GOOGLE_OAUTH_ENABLED).toBe(false);
        expect(GOOGLE_CLIENT_ID).toBe("");
    });

    it("fails fast when Google OAuth is enabled without a client id", async () => {
        vi.stubEnv("VITE_API_ORIGIN", "http://localhost:8000");
        vi.stubEnv("VITE_API_PATH", "/api/v0");
        vi.stubEnv("VITE_GOOGLE_OAUTH_ENABLED", "true");
        vi.stubEnv("VITE_GOOGLE_CLIENT_ID", "");

        await expect(import("./config")).rejects.toThrow(
            "Incomplete Google OAuth frontend config. Set googleClientId when googleOAuthEnabled=true.",
        );
    });
});
