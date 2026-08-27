import { API_URL } from "../configs/config";

export type AuthMode = "required" | "none";

interface ApiFetchOptions {
    authMode?: AuthMode;
    retryOnAuthFailure?: boolean;
}

type AuthFailureHandler = (() => void) | null;

let refreshPromise: Promise<boolean> | null = null;
let authFailureHandler: AuthFailureHandler = null;
let csrfToken: string | null = null;
let csrfPromise: Promise<string | null> | null = null;

export function setApiAuthFailureHandler(handler: AuthFailureHandler) {
    authFailureHandler = handler;
}

async function refreshAccessToken() {
    if (!refreshPromise) {
        refreshPromise = csrfFetch(`${API_URL}/auth/refresh`, {
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

function isMutatingMethod(method?: string) {
    const normalizedMethod = (method ?? "GET").toUpperCase();
    return ["POST", "PUT", "PATCH", "DELETE"].includes(normalizedMethod);
}

async function ensureCSRFToken() {
    if (csrfToken) {
        return csrfToken;
    }

    if (!csrfPromise) {
        csrfPromise = fetch(`${API_URL}/auth/csrf`, {
            method: "GET",
            credentials: "include",
        })
            .then(async (response) => {
                if (!response.ok) {
                    return null;
                }

                const data = (await response.json()) as { csrfToken?: string };
                csrfToken = typeof data.csrfToken === "string" ? data.csrfToken : null;
                return csrfToken;
            })
            .catch(() => null)
            .finally(() => {
                csrfPromise = null;
            });
    }

    return csrfPromise;
}

async function csrfFetch(input: string, init: RequestInit = {}) {
    const headers = new Headers(init.headers);

    if (isMutatingMethod(init.method)) {
        const token = await ensureCSRFToken();
        if (token) {
            headers.set("X-CSRF-Token", token);
        }
    }

    return fetch(input, {
        ...init,
        headers,
    });
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

    const response = await csrfFetch(`${API_URL}${path}`, {
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
    return (await getResponseError(response, fallback)).message;
}

export interface ResponseError {
    message: string;
    code: string | null;
}

export async function getResponseError(
    response: Response,
    fallback: string,
): Promise<ResponseError> {
    const contentType = response.headers.get("content-type") ?? "";

    try {
        if (contentType.includes("application/json")) {
            const data = (await response.json()) as unknown;
            if (typeof data !== "object" || data === null) {
                return { message: fallback, code: null };
            }

            const errorData = data as {
                error?: unknown;
                message?: unknown;
                code?: unknown;
            };
            const code =
                typeof errorData.code === "string" && errorData.code.trim()
                    ? errorData.code.trim()
                    : null;
            if (typeof errorData.error === "string") {
                const error = errorData.error.trim();
                if (error) return { message: error, code };
            }
            if (typeof errorData.message === "string") {
                const message = errorData.message.trim();
                if (message) return { message, code };
            }
            return { message: fallback, code };
        }

        const text = (await response.text()).trim();
        return { message: text || fallback, code: null };
    } catch {
        return { message: fallback, code: null };
    }
}
