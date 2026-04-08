const GOOGLE_IDENTITY_SCRIPT_ID = "google-identity-services-client";
const GOOGLE_IDENTITY_SCRIPT_SRC = "https://accounts.google.com/gsi/client";

let googleIdentityPromise: Promise<GoogleAccountsId> | null = null;

function getGoogleAccountsId() {
    return window.google?.accounts?.id;
}

export function loadGoogleAccountsId() {
    if (typeof window === "undefined" || typeof document === "undefined") {
        return Promise.reject(
            new Error("Google sign-in requires a browser environment.")
        );
    }

    const existingGoogleAccountsId = getGoogleAccountsId();
    if (existingGoogleAccountsId) {
        return Promise.resolve(existingGoogleAccountsId);
    }

    if (googleIdentityPromise) {
        return googleIdentityPromise;
    }

    googleIdentityPromise = new Promise<GoogleAccountsId>((resolve, reject) => {
        const handleLoad = () => {
            const googleAccountsId = getGoogleAccountsId();
            if (googleAccountsId) {
                resolve(googleAccountsId);
                return;
            }

            googleIdentityPromise = null;
            reject(
                new Error(
                    "Google sign-in SDK loaded, but the GIS API was unavailable."
                )
            );
        };

        const handleError = () => {
            googleIdentityPromise = null;
            reject(new Error("Failed to load the Google sign-in SDK."));
        };

        const existingScript = document.getElementById(
            GOOGLE_IDENTITY_SCRIPT_ID
        ) as HTMLScriptElement | null;

        if (existingScript) {
            existingScript.addEventListener("load", handleLoad, { once: true });
            existingScript.addEventListener("error", handleError, {
                once: true,
            });
            return;
        }

        const script = document.createElement("script");
        script.id = GOOGLE_IDENTITY_SCRIPT_ID;
        script.src = GOOGLE_IDENTITY_SCRIPT_SRC;
        script.async = true;
        script.defer = true;
        script.addEventListener("load", handleLoad, { once: true });
        script.addEventListener("error", handleError, { once: true });

        document.head.append(script);
    });

    return googleIdentityPromise;
}
