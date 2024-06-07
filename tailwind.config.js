/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [    "./frontend/views/**/*.{html,templ,go}"],
  theme: {
    extend: {},
  },
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: ["dark", "cupcake"],
  },
}