import { beforeEach, describe, expect, it } from "vitest";

import {
    claimPWAUpdateActivation,
    hasRemotePWAUpdateBlocker,
    PWA_ACTIVATION_LEASE_MS,
    PWA_TAB_STATE_TTL_MS,
    releasePWAUpdateActivation,
    removePWAUpdateTabState,
    writePWAUpdateTabState,
} from "./pwaUpdateCoordination";

describe("PWA update coordination", () => {
    beforeEach(() => localStorage.clear());

    it("reports blockers from another live tab without storing form data", () => {
        expect(writePWAUpdateTabState(localStorage, "tab-a", true, 100)).toBe(true);

        expect(hasRemotePWAUpdateBlocker(localStorage, "tab-b", 101)).toBe(true);
        expect(localStorage.getItem("expense-tracker:pwa-update:tab:tab-a"))
            .toBe(JSON.stringify({ blocked: true, updatedAt: 100 }));
    });

    it("removes stale tab records so a closed tab cannot block forever", () => {
        writePWAUpdateTabState(localStorage, "tab-a", true, 100);

        expect(
            hasRemotePWAUpdateBlocker(
                localStorage,
                "tab-b",
                100 + PWA_TAB_STATE_TTL_MS + 1,
            ),
        ).toBe(false);
        expect(localStorage.length).toBe(0);
    });

    it("allows only one live activation lease and recovers after expiry", () => {
        const firstToken = claimPWAUpdateActivation(localStorage, "tab-a", 100);
        expect(firstToken).not.toBeNull();
        expect(claimPWAUpdateActivation(localStorage, "tab-b", 101)).toBeNull();

        const secondToken = claimPWAUpdateActivation(
            localStorage,
            "tab-b",
            100 + PWA_ACTIVATION_LEASE_MS + 1,
        );
        expect(secondToken).not.toBeNull();
    });

    it("only lets the lease owner release it", () => {
        const token = claimPWAUpdateActivation(localStorage, "tab-a", 100);
        expect(token).not.toBeNull();

        releasePWAUpdateActivation(localStorage, "tab-b", token ?? "");
        expect(claimPWAUpdateActivation(localStorage, "tab-c", 101)).toBeNull();

        releasePWAUpdateActivation(localStorage, "tab-a", token ?? "");
        expect(claimPWAUpdateActivation(localStorage, "tab-c", 102)).not.toBeNull();
    });

    it("cleans up a tab record on a normal page exit", () => {
        writePWAUpdateTabState(localStorage, "tab-a", false, 100);
        removePWAUpdateTabState(localStorage, "tab-a");
        expect(localStorage.length).toBe(0);
    });
});
