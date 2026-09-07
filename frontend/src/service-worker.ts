/// <reference lib="webworker" />
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from "workbox-precaching";
import { NavigationRoute, registerRoute } from "workbox-routing";
import { NetworkFirst } from "workbox-strategies";
import { ExpirationPlugin } from "workbox-expiration";

declare const self: ServiceWorkerGlobalScope & typeof globalThis & {
    __WB_MANIFEST: Array<unknown>;
};

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);
registerRoute(new NavigationRoute(createHandlerBoundToURL("index.html"), { denylist: [/^\/api\//, /^\/auth\//] }));
registerRoute(({ url }) => url.pathname.endsWith("/runtime-config.js"), new NetworkFirst({ cacheName: "runtime-config", networkTimeoutSeconds: 3, plugins: [new ExpirationPlugin({ maxEntries: 1, maxAgeSeconds: 60 * 60 * 24 })] }));
self.addEventListener("push", (event) => {
    const payload = event.data?.json() as { title?: string; body?: string; tag?: string } | undefined;
    event.waitUntil(self.registration.showNotification(payload?.title ?? "Expense Tracker", { body: payload?.body ?? "New activity is available", tag: payload?.tag ?? "expense-tracker-activity", data: {} }));
});
self.addEventListener("notificationclick", (event) => {
    event.notification.close();
    event.waitUntil(self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((windows) => windows[0]?.focus() ?? self.clients.openWindow("/")));
});
