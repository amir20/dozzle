import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";

export default {
  content: ["./assets/**/*.{vue,js,ts}", "./public/index.html"],
  theme: {
    extend: {
      animation: {
        "bounce-fast": "bounce 0.5s 2 both",
      },
      colors: {
        turquoise: "hsl(171, 100%, 41%)",
        yellow: "hsl(44,  100%, 77%)",
        black: "hsl(0, 0%, 4%)",
        "black-bis": "hsl(0, 0%, 7%)",
        "black-ter": "hsl(0, 0%, 14%)",
        "grey-darker": "hsl(0, 0%, 21%)",
        "grey-dark": "hsl(0, 0%, 29%)",
        grey: "hsl(0, 0%, 48%)",
        "grey-light": "hsl(0, 0%, 71%)",
        "grey-lighter": "hsl(0, 0%, 86%)",
        "grey-lightest": "hsl(0, 0%, 93%)",
        "white-ter": "hsl(0, 0%, 96%)",
        "white-bis": "hsl(0, 0%, 98%)",
        white: "hsl(0, 0%, 100%)",
        "scheme-bis": "var(--scheme-main-bis)",
        green: "#00b5ad",
        red: "#f44336",
        purple: "#9c27b0",
        orange: "#ff9800",
        blue: "#2196f3",
        "scheme-main": "var(--scheme-main-color)",
        "scheme-main-bis": "var(--scheme-main-bis-color)",
        "scheme-main-ter": "var(--scheme-main-ter-color)",
        "scheme-inverted": "var(--scheme-inverted-color)",
        "scheme-inverted-bis": "var(--scheme-inverted-bis-color)",
        "scheme-inverted-ter": "var(--scheme-inverted-ter-color)",
        primary: "var(--primary-color)",
        secondary: "var(--secondary-color)",
      },
    },
  },
  plugins: [DaisyUI],
  daisyui: {
    themes: [],
    base: false,
    // styled: false,
  },
} satisfies Config;
