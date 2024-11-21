/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./frontend/views/**/*.{html,templ,go}"],
    theme: {
        extend: {
            colors: {
                google: {
                    "text-gray": "#3c4043",
                    "button-blue": "#1a73e8",
                    "button-blue-hover": "#5195ee",
                    "button-dark": "#202124",
                    "button-dark-hover": "#555658",
                    "button-border-light": "#dadce0",
                    "logo-blue": "#4285f4",
                    "logo-green": "#34a853",
                    "logo-yellow": "#fbbc05",
                    "logo-red": "#ea4335",
                },
            },

            // that is animation class
            animation: {
                fade: "fadeOut 5s ease-in-out forwards",
            },

            // that is actual animation
            keyframes: (theme) => ({
                fadeOut: {
                    "0%": { opacity: "1" },
                    "100%": { opacity: "0" },
                },
            }),
        },
    },
    plugins: [require("daisyui")],
    daisyui: {
        themes: ["dark", "cupcake"],
    },
};
