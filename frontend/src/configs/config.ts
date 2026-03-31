type AppRuntimeConfig = {
    apiOrigin: string;
    apiPath: string;
};

const DEFAULT_RUNTIME_CONFIG: AppRuntimeConfig = {
    apiOrigin: "http://localhost:8080",
    apiPath: "/api/v0",
};

function isLocalDevelopmentRuntime() {
    return ["localhost", "127.0.0.1"].includes(window.location.hostname);
}

function getRuntimeConfig(): AppRuntimeConfig {
    if (typeof window === "undefined" || isLocalDevelopmentRuntime()) {
        return DEFAULT_RUNTIME_CONFIG;
    }

    const apiOrigin = window.__APP_CONFIG__?.apiOrigin?.trim();
    const apiPath = window.__APP_CONFIG__?.apiPath?.trim();

    if (!apiOrigin || !apiPath) {
        throw new Error(
            "Missing runtime API config. Ensure /runtime-config.js defines apiOrigin and apiPath."
        );
    }

    return {
        apiOrigin,
        apiPath,
    };
}

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

const runtimeConfig = getRuntimeConfig();

export const API_URL = `${normalizeOrigin(runtimeConfig.apiOrigin)}${runtimeConfig.apiPath}`;
