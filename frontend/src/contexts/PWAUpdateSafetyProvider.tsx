import {
    ReactNode,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState,
} from "react";

import { subscribeMutationActivity } from "../lib/mutationActivity";
import {
    claimPWAUpdateActivation,
    createPWAUpdateID,
    hasRemotePWAUpdateBlocker,
    PWA_TAB_HEARTBEAT_MS,
    releasePWAUpdateActivation,
    removePWAUpdateTabState,
    writePWAUpdateTabState,
} from "../lib/pwaUpdateCoordination";
import { PWAUpdateSafetyContext } from "./PWAUpdateSafetyContext";

export function PWAUpdateSafetyProvider({ children }: { children: ReactNode }) {
    const tabID = useRef(createPWAUpdateID());
    const blockers = useRef(new Set<string>());
    const leaseToken = useRef<string | null>(null);
    const localBlocker = useRef(false);
    const [hasLocalBlocker, setHasLocalBlocker] = useState(false);
    const [hasRemoteBlocker, setHasRemoteBlocker] = useState(false);
    const [canAutoApply, setCanAutoApply] = useState(false);

    const registerBlocker = useCallback((id: string, blocked: boolean) => {
        if (blocked) blockers.current.add(id);
        else blockers.current.delete(id);
        const nextHasLocalBlocker = blockers.current.size > 0;
        localBlocker.current = nextHasLocalBlocker;
        setHasLocalBlocker(nextHasLocalBlocker);
    }, []);

    useEffect(() => subscribeMutationActivity((active) => {
        registerBlocker("api-mutation", active);
    }), [registerBlocker]);

    const refreshCoordination = useCallback(() => {
        let storage: Storage;
        try {
            storage = window.localStorage;
        } catch {
            setCanAutoApply(false);
            setHasRemoteBlocker(true);
            return;
        }
        const available = writePWAUpdateTabState(
            storage,
            tabID.current,
            localBlocker.current,
        );
        setCanAutoApply(available);
        setHasRemoteBlocker(
            available
                ? hasRemotePWAUpdateBlocker(storage, tabID.current)
                : true,
        );
    }, []);

    useEffect(() => {
        refreshCoordination();
    }, [hasLocalBlocker, refreshCoordination]);

    useEffect(() => {
        const currentTabID = tabID.current;
        const handleStorage = () => refreshCoordination();
        const handlePageHide = (event: PageTransitionEvent) => {
            if (event.persisted) return;
            try {
                removePWAUpdateTabState(window.localStorage, currentTabID);
            } catch {
                // Stale records expire if browser storage is unavailable.
            }
        };
        window.addEventListener("storage", handleStorage);
        window.addEventListener("pagehide", handlePageHide);
        const heartbeat = window.setInterval(
            refreshCoordination,
            PWA_TAB_HEARTBEAT_MS,
        );
        return () => {
            window.removeEventListener("storage", handleStorage);
            window.removeEventListener("pagehide", handlePageHide);
            window.clearInterval(heartbeat);
            try {
                removePWAUpdateTabState(window.localStorage, currentTabID);
            } catch {
                // Stale records expire if browser storage is unavailable.
            }
        };
    }, [refreshCoordination]);

    const claimActivation = useCallback(() => {
        if (!canAutoApply || hasLocalBlocker) return false;
        let storage: Storage;
        try {
            storage = window.localStorage;
        } catch {
            setCanAutoApply(false);
            return false;
        }
        if (hasRemotePWAUpdateBlocker(storage, tabID.current)) {
            setHasRemoteBlocker(true);
            return false;
        }
        const token = claimPWAUpdateActivation(
            storage,
            tabID.current,
        );
        leaseToken.current = token;
        return token !== null;
    }, [canAutoApply, hasLocalBlocker]);

    const releaseActivation = useCallback(() => {
        if (!leaseToken.current) return;
        try {
            releasePWAUpdateActivation(
                window.localStorage,
                tabID.current,
                leaseToken.current,
            );
        } catch {
            // The short lease expiry recovers if storage becomes unavailable.
        }
        leaseToken.current = null;
    }, []);

    const value = useMemo(() => ({
        canAutoApply,
        isUpdateSafe: !hasLocalBlocker && !hasRemoteBlocker,
        registerBlocker,
        claimActivation,
        releaseActivation,
    }), [
        canAutoApply,
        claimActivation,
        hasLocalBlocker,
        hasRemoteBlocker,
        registerBlocker,
        releaseActivation,
    ]);

    return (
        <PWAUpdateSafetyContext.Provider value={value}>
            {children}
        </PWAUpdateSafetyContext.Provider>
    );
}
