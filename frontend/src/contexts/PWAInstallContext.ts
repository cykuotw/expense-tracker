import { createContext } from "react";

export interface PWAInstallContextValue {
    canInstall: boolean;
    isIOS: boolean;
    requestInstall: () => Promise<void>;
}

export const PWAInstallContext = createContext<PWAInstallContextValue>({
    canInstall: false,
    isIOS: false,
    requestInstall: async () => undefined,
});
