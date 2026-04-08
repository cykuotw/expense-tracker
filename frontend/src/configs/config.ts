type AppConfig = {
    apiOrigin: string;
    apiPath: string;
    googleOAuthEnabled: boolean;
    googleClientId: string;
};

const getAppConfig = (): AppConfig => {
    const source = import.meta.env.DEV ? "local env" : "runtime-config.js";

    const rawConfig = import.meta.env.DEV
        ? {
              apiOrigin: import.meta.env.VITE_API_ORIGIN,
              apiPath: import.meta.env.VITE_API_PATH,
              googleOAuthEnabled: import.meta.env.VITE_GOOGLE_OAUTH_ENABLED,
              googleClientId: import.meta.env.VITE_GOOGLE_CLIENT_ID,
          }
        : (() => {
              if (typeof window === "undefined") {
                  throw new Error("Frontend config requires a browser window.");
              }

              return {
                  apiOrigin: window.__APP_CONFIG__?.apiOrigin,
                  apiPath: window.__APP_CONFIG__?.apiPath,
                  googleOAuthEnabled: window.__APP_CONFIG__?.googleOAuthEnabled,
                  googleClientId: window.__APP_CONFIG__?.googleClientId,
              };
          })();

    const requireString = (value: string | undefined, key: string) => {
        const normalizedValue = value?.trim();
        if (!normalizedValue) {
            throw new Error(`Missing frontend config: ${key}`);
        }

        return normalizedValue;
    };

    const parseBoolean = (
        value: string | boolean | undefined,
        key: string,
        defaultValue = false,
    ) => {
        if (typeof value === "boolean") {
            return value;
        }

        if (value === undefined) {
            return defaultValue;
        }

        const normalizedValue = value.trim().toLowerCase();
        if (normalizedValue === "") {
            return defaultValue;
        }

        if (normalizedValue === "true") {
            return true;
        }

        if (normalizedValue === "false") {
            return false;
        }

        throw new Error(
            `Invalid frontend config for ${key}. Expected "true" or "false".`,
        );
    };

    const config = {
        apiOrigin: requireString(rawConfig.apiOrigin, `${source} apiOrigin`),
        apiPath: requireString(rawConfig.apiPath, `${source} apiPath`),
        googleOAuthEnabled: parseBoolean(
            rawConfig.googleOAuthEnabled,
            `${source} googleOAuthEnabled`,
        ),
        googleClientId: rawConfig.googleClientId?.trim() ?? "",
    };

    if (config.googleOAuthEnabled && !config.googleClientId) {
        throw new Error(
            "Incomplete Google OAuth frontend config. Set googleClientId when googleOAuthEnabled=true.",
        );
    }

    return config;
};

const normalizeOrigin = (value?: string) => {
    if (!value) {
        return "";
    }

    if (value.startsWith("http://") || value.startsWith("https://")) {
        return value;
    }

    const protocol =
        globalThis.window?.location.protocol === "https:" ? "https:" : "http:";

    return `${protocol}//${value}`;
};

export const APP_CONFIG = getAppConfig();

export const API_URL = `${normalizeOrigin(APP_CONFIG.apiOrigin)}${APP_CONFIG.apiPath}`;
export const GOOGLE_OAUTH_ENABLED = APP_CONFIG.googleOAuthEnabled;
export const GOOGLE_CLIENT_ID = APP_CONFIG.googleClientId;
