/// <reference lib="webworker" />
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from "workbox-precaching";
import { NavigationRoute, registerRoute } from "workbox-routing";
import { NetworkFirst, NetworkOnly } from "workbox-strategies";
import { ExpirationPlugin } from "workbox-expiration";

declare const self: ServiceWorkerGlobalScope & typeof globalThis & {
    __WB_MANIFEST: Array<unknown>;
};

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);

// vite-plugin-pwa's prompt registration sends this message when the user
// chooses “Update now”. injectManifest workers must handle it themselves;
// without it, the downloaded worker remains waiting and the current shell
// continues to serve the previous application version.
self.addEventListener("message", (event) => {
    if (event.data?.type === "SKIP_WAITING") {
        event.waitUntil(self.skipWaiting());
    }
});

self.addEventListener("activate", (event) => {
    event.waitUntil(self.clients.claim());
});

// Signed receipt URLs remain outside both Workbox and browser HTTP caches.
// This fetch also requires the receipt origin in the worker's CSP connect-src.
registerRoute(({ url }) => url.searchParams.has("X-Amz-Signature"), new NetworkOnly({ fetchOptions: { cache: "no-store" } }));
registerRoute(({ url }) => url.pathname.startsWith("/api/"), new NetworkOnly());
registerRoute(({ url }) => url.pathname === "/operating-hours.json", new NetworkOnly({ fetchOptions: { cache: "no-store" } }));
registerRoute(new NavigationRoute(createHandlerBoundToURL("index.html"), { denylist: [/^\/api\//, /^\/auth\//] }));
registerRoute(({ url }) => url.pathname.endsWith("/runtime-config.js"), new NetworkFirst({ cacheName: "runtime-config", networkTimeoutSeconds: 3, plugins: [new ExpirationPlugin({ maxEntries: 1, maxAgeSeconds: 60 * 60 * 24 })] }));
