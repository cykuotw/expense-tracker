import { API_URL } from "../configs/config";

export type AuthMode = "required" | "none";

interface ApiFetchOptions {
    authMode?: AuthMode;
    retryOnAuthFailure?: boolean;
}

type AuthFailureHandler = (() => void) | null;

let refreshPromise: Promise<boolean> | null = null;
let authFailureHandler: AuthFailureHandler = null;

export function setApiAuthFailureHandler(handler: AuthFailureHandler) {
    authFailureHandler = handler;
}

async function refreshAccessToken() {
    if (!refreshPromise) {
        refreshPromise = fetch(`${API_URL}/auth/refresh`, {
            method: "POST",
            credentials: "include",
        })
            .then((response) => response.ok)
            .catch(() => false)
            .finally(() => {
                refreshPromise = null;
            });
    }

    return refreshPromise;
}

export async function apiFetch(
    path: string,
    init: RequestInit = {},
    options: ApiFetchOptions = {}
) {
    const { authMode = "required", retryOnAuthFailure = true } = options;
    const headers = new Headers(init.headers);

    if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
    }

    const response = await fetch(`${API_URL}${path}`, {
        ...init,
        credentials: "include",
        headers,
    });

    if (authMode !== "required" || response.status !== 401) {
        return response;
    }

    if (!retryOnAuthFailure) {
        authFailureHandler?.();
        return response;
    }

    const refreshed = await refreshAccessToken();
    if (!refreshed) {
        authFailureHandler?.();
        return response;
    }

    return apiFetch(path, init, { ...options, retryOnAuthFailure: false });
}

export async function getResponseErrorMessage(
    response: Response,
    fallback: string
) {
    const contentType = response.headers.get("content-type") ?? "";

    try {
        if (contentType.includes("application/json")) {
            const data = (await response.json()) as {
                error?: string;
                message?: string;
            };
            if (typeof data.error === "string" && data.error.length > 0) {
                return data.error;
            }
            if (typeof data.message === "string" && data.message.length > 0) {
                return data.message;
            }
            return fallback;
        }

        const text = await response.text();
        return text || fallback;
    } catch {
        return fallback;
    }
}
