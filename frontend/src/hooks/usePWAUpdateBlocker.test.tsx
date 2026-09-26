import { renderHook } from "@testing-library/react";
import { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import {
    PWAUpdateSafetyContext,
    PWAUpdateSafetyContextValue,
} from "../contexts/PWAUpdateSafetyContext";
import { usePWAUpdateBlocker } from "./usePWAUpdateBlocker";

describe("usePWAUpdateBlocker", () => {
    it("updates its blocker and clears it on unmount", () => {
        const registerBlocker = vi.fn();
        const value: PWAUpdateSafetyContextValue = {
            canAutoApply: true,
            isUpdateSafe: true,
            registerBlocker,
            claimActivation: () => true,
            releaseActivation: () => undefined,
        };
        const wrapper = ({ children }: { children: ReactNode }) => (
            <PWAUpdateSafetyContext.Provider value={value}>
                {children}
            </PWAUpdateSafetyContext.Provider>
        );

        const { rerender, unmount } = renderHook(
            ({ blocked }) => usePWAUpdateBlocker(blocked),
            { initialProps: { blocked: true }, wrapper },
        );
        const blockerID = registerBlocker.mock.calls[0]?.[0] as string;
        expect(registerBlocker).toHaveBeenLastCalledWith(blockerID, true);

        rerender({ blocked: false });
        expect(registerBlocker).toHaveBeenLastCalledWith(blockerID, false);

        unmount();
        expect(registerBlocker).toHaveBeenLastCalledWith(blockerID, false);
    });
});
