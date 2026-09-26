import { createContext } from "react";

export interface PWAUpdateSafetyContextValue {
    canAutoApply: boolean;
    isUpdateSafe: boolean;
    registerBlocker: (id: string, blocked: boolean) => void;
    claimActivation: () => boolean;
    releaseActivation: () => void;
}

export const PWAUpdateSafetyContext =
    createContext<PWAUpdateSafetyContextValue>({
        canAutoApply: false,
        isUpdateSafe: false,
        registerBlocker: () => undefined,
        claimActivation: () => false,
        releaseActivation: () => undefined,
    });
