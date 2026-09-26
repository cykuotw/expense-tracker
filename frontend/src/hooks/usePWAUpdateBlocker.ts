import { useEffect, useId } from "react";

import { usePWAUpdateSafety } from "./usePWAUpdateSafety";

export function usePWAUpdateBlocker(blocked: boolean) {
    const id = useId();
    const { registerBlocker } = usePWAUpdateSafety();

    useEffect(() => {
        registerBlocker(id, blocked);
        return () => registerBlocker(id, false);
    }, [blocked, id, registerBlocker]);
}
