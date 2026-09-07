import { defineConfig } from "vitest/config";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";
import tsconfigPaths from "vite-tsconfig-paths";

// https://vite.dev/config/
export default defineConfig({
    plugins: [
        tailwindcss(),
        ...(process.env.VITEST ? [] : [react()]),
        tsconfigPaths(),
        VitePWA({
			strategies: "injectManifest",
			srcDir: "src",
			filename: "service-worker.ts",
            registerType: "prompt",
            injectRegister: false,
            manifest: {
                name: "Expense Tracker",
                short_name: "Expenses",
                description: "Track shared expenses and balances.",
                theme_color: "#1f4d47",
                background_color: "#fffdf8",
                display: "standalone",
                start_url: "/",
                scope: "/",
                icons: [
                    {
                        src: "/pwa-64x64.png",
                        sizes: "64x64",
                        type: "image/png",
                    },
                    {
                        src: "/pwa-192x192.png",
                        sizes: "192x192",
                        type: "image/png",
                    },
                    {
                        src: "/pwa-512x512.png",
                        sizes: "512x512",
                        type: "image/png",
                        purpose: "any",
                    },
                    {
                        src: "/maskable-icon-512x512.png",
                        sizes: "512x512",
                        type: "image/png",
                        purpose: "maskable",
                    },
                ],
            },
            injectManifest: {
                globIgnores: ["pwa-icon-source.png", "runtime-config.js"],
            },
        }),
    ],
    build: {
        rollupOptions: {
            onwarn(warning, warn) {
                if (
                    warning.code === "MODULE_LEVEL_DIRECTIVE" &&
                    (warning.id?.includes("@radix-ui/") ||
                        warning.id?.includes("react-hot-toast"))
                ) {
                    return;
                }

                warn(warning);
            },
        },
    },
    test: {
        environment: "jsdom",
        setupFiles: "./src/test/setup.ts",
        css: true,
    },
});
