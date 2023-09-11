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
        turquoise: {
          DEFAULT: "hsl(171, 100%, 41%)",
          dark: "hsl(171, 100%, 31%)",
          light: "hsl(171, 100%, 71%)",
        },
        yellow: "hsl(44,  100%, 77%)",
        orange: "hsl(25, 95%, 53%)",
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
        green: "hsl(177, 100%, 35%)",
        red: "hsl(4, 90%, 58%)",
        purple: "hsl(291, 64%, 42%)",
        blue: "hsl(207, 90%, 54%)",
        // theme
        "scheme-main": "var(--scheme-main-color)",
        "scheme-main-bis": "var(--scheme-main-bis-color)",
        "scheme-main-ter": "var(--scheme-main-ter-color)",
        "scheme-inverted": "var(--scheme-inverted-color)",
        "scheme-inverted-bis": "var(--scheme-inverted-bis-color)",
        "scheme-inverted-ter": "var(--scheme-inverted-ter-color)",
        primary: "var(--primary-color)",
        secondary: "var(--secondary-color)",
        neutural: "var(--accent-color)",
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
