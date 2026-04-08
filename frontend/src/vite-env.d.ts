/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_API_ORIGIN?: string;
    readonly VITE_API_PATH?: string;
    readonly VITE_GOOGLE_OAUTH_ENABLED?: string;
    readonly VITE_GOOGLE_CLIENT_ID?: string;
}

interface ImportMeta {
    readonly env: ImportMetaEnv;
}

interface Window {
    __APP_CONFIG__?: {
        apiOrigin?: string;
        apiPath?: string;
        googleOAuthEnabled?: boolean | string;
        googleClientId?: string;
    };
}
