import { useContext } from "react";

import { PWAUpdateSafetyContext } from "../contexts/PWAUpdateSafetyContext";

export function usePWAUpdateSafety() {
    return useContext(PWAUpdateSafetyContext);
}
