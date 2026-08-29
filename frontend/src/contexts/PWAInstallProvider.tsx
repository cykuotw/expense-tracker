import { ReactNode, useCallback, useEffect, useState } from "react";

import { PWAInstallContext } from "./PWAInstallContext";

interface BeforeInstallPromptEvent extends Event {
    prompt: () => Promise<void>;
}

function isIOSDevice() {
    return /iPad|iPhone|iPod/.test(navigator.userAgent) ||
        (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
}

function isStandaloneDisplayMode() {
    const navigatorWithStandalone = navigator as Navigator & {
        standalone?: boolean;
    };
    return (
        navigatorWithStandalone.standalone === true ||
        window.matchMedia?.("(display-mode: standalone)").matches === true
    );
}

export function PWAInstallProvider({ children }: { children: ReactNode }) {
    const [deferredPrompt, setDeferredPrompt] =
        useState<BeforeInstallPromptEvent | null>(null);
    const [installed, setInstalled] = useState(isStandaloneDisplayMode);
    const [isIOS] = useState(isIOSDevice);

    useEffect(() => {
        const mediaQuery = window.matchMedia?.("(display-mode: standalone)");
        const updateInstalled = () => setInstalled(isStandaloneDisplayMode());
        const captureInstallPrompt = (event: Event) => {
            event.preventDefault();
            setDeferredPrompt(event as BeforeInstallPromptEvent);
        };
        const handleAppInstalled = () => {
            setDeferredPrompt(null);
            setInstalled(true);
        };

        window.addEventListener("beforeinstallprompt", captureInstallPrompt);
        window.addEventListener("appinstalled", handleAppInstalled);
        mediaQuery?.addEventListener("change", updateInstalled);
        updateInstalled();

        return () => {
            window.removeEventListener(
                "beforeinstallprompt",
                captureInstallPrompt,
            );
            window.removeEventListener("appinstalled", handleAppInstalled);
            mediaQuery?.removeEventListener("change", updateInstalled);
        };
    }, []);

    const requestInstall = useCallback(async () => {
        if (!deferredPrompt) return;

        try {
            await deferredPrompt.prompt();
        } finally {
            setDeferredPrompt(null);
        }
    }, [deferredPrompt]);

    return (
        <PWAInstallContext.Provider
            value={{
                canInstall: !installed && (isIOS || deferredPrompt !== null),
                isIOS,
                requestInstall,
            }}
        >
            {children}
        </PWAInstallContext.Provider>
    );
}
