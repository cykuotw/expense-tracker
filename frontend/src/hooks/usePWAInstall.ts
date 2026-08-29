import { useContext } from "react";

import { PWAInstallContext } from "../contexts/PWAInstallContext";

export function usePWAInstall() {
    return useContext(PWAInstallContext);
}
