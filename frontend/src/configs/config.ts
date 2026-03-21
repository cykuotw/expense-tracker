const API_ORIGIN = import.meta.env.VITE_API_ORIGIN;
const API_PATH = import.meta.env.VITE_API_PATH ?? "";

function getBrowserProtocol() {
    if (typeof window === "undefined") {
        return "http:";
    }

    return window.location.protocol === "https:" ? "https:" : "http:";
}

function normalizeOrigin(value?: string) {
    if (!value) {
        return "";
    }

    if (value.startsWith("http://") || value.startsWith("https://")) {
        return value;
    }

    return `${getBrowserProtocol()}//${value}`;
}

export const API_URL = `${normalizeOrigin(API_ORIGIN)}${API_PATH}`;
