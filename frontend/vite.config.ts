import { defineConfig } from "vitest/config";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
    plugins: [tailwindcss(), ...(process.env.VITEST ? [] : [react()])],
    build: {
        rollupOptions: {
            onwarn(warning, warn) {
                if (
                    warning.message.includes('"use client"') &&
                    warning.id?.includes("react-hot-toast")
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
