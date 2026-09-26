const TAB_STATE_PREFIX = "expense-tracker:pwa-update:tab:";
const ACTIVATION_LEASE_KEY = "expense-tracker:pwa-update:activation";

export const PWA_TAB_STATE_TTL_MS = 24 * 60 * 60 * 1000;
export const PWA_TAB_HEARTBEAT_MS = 30_000;
export const PWA_ACTIVATION_LEASE_MS = 15_000;

interface TabState {
    blocked: boolean;
    updatedAt: number;
}

interface ActivationLease {
    owner: string;
    token: string;
    expiresAt: number;
}

function parseRecord<T>(value: string | null): T | null {
    if (!value) return null;
    try {
        return JSON.parse(value) as T;
    } catch {
        return null;
    }
}

export function createPWAUpdateID() {
    return globalThis.crypto?.randomUUID?.() ??
        `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function writePWAUpdateTabState(
    storage: Storage,
    tabID: string,
    blocked: boolean,
    now = Date.now(),
) {
    try {
        const key = `${TAB_STATE_PREFIX}${tabID}`;
        storage.setItem(key, JSON.stringify({ blocked, updatedAt: now }));
        return storage.getItem(key) !== null;
    } catch {
        return false;
    }
}

export function removePWAUpdateTabState(storage: Storage, tabID: string) {
    try {
        storage.removeItem(`${TAB_STATE_PREFIX}${tabID}`);
    } catch {
        // Best-effort cleanup; stale records expire shortly.
    }
}

export function hasRemotePWAUpdateBlocker(
    storage: Storage,
    ownTabID: string,
    now = Date.now(),
) {
    let blocked = false;
    try {
        for (let index = storage.length - 1; index >= 0; index -= 1) {
            const key = storage.key(index);
            if (!key?.startsWith(TAB_STATE_PREFIX)) continue;

            const tabID = key.slice(TAB_STATE_PREFIX.length);
            const state = parseRecord<TabState>(storage.getItem(key));
            if (
                !state ||
                typeof state.blocked !== "boolean" ||
                typeof state.updatedAt !== "number" ||
                now - state.updatedAt > PWA_TAB_STATE_TTL_MS
            ) {
                storage.removeItem(key);
                continue;
            }
            if (tabID !== ownTabID && state.blocked) blocked = true;
        }
        return blocked;
    } catch {
        return true;
    }
}

export function claimPWAUpdateActivation(
    storage: Storage,
    tabID: string,
    now = Date.now(),
) {
    try {
        const current = parseRecord<ActivationLease>(
            storage.getItem(ACTIVATION_LEASE_KEY),
        );
        if (current && current.expiresAt > now && current.owner !== tabID) {
            return null;
        }

        const token = createPWAUpdateID();
        const lease: ActivationLease = {
            owner: tabID,
            token,
            expiresAt: now + PWA_ACTIVATION_LEASE_MS,
        };
        storage.setItem(ACTIVATION_LEASE_KEY, JSON.stringify(lease));
        const stored = parseRecord<ActivationLease>(
            storage.getItem(ACTIVATION_LEASE_KEY),
        );
        return stored?.owner === tabID && stored.token === token ? token : null;
    } catch {
        return null;
    }
}

export function releasePWAUpdateActivation(
    storage: Storage,
    tabID: string,
    token: string,
) {
    try {
        const current = parseRecord<ActivationLease>(
            storage.getItem(ACTIVATION_LEASE_KEY),
        );
        if (current?.owner === tabID && current.token === token) {
            storage.removeItem(ACTIVATION_LEASE_KEY);
        }
    } catch {
        // The short lease expiry recovers if storage becomes unavailable.
    }
}
