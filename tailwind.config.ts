import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";
import Typography from "@tailwindcss/typography";

export default {
  content: ["./assets/**/*.{vue,js,ts}", "./public/index.html"],
  theme: {
    extend: {
      animation: {
        "bounce-fast": "bounce 0.5s 2 both",
      },
      colors: {
        green: "hsl(177 100% 35%)",
        red: "hsl(4 90% 58%)",
        purple: "hsl(291 64% 42%)",
        blue: "hsl(207 90% 54%)",
        orange: "hsl(25 95% 53%)",
        base: "hsl(var(--base-color) / <alpha-value>)",
        "base-darker": "hsl(var(--base-darker-color) / <alpha-value>)",
        "base-lighter": "hsl(var(--base-lighter-color) / <alpha-value>)",
        "base-content": "hsl(var(--base-content-color) / <alpha-value>)",

        primary: "hsl(var(--primary-color) / <alpha-value>)",
        "primary-focus": "hsl(var(--primary-focus-color) / <alpha-value>)",
        secondary: "hsl(var(--secondary-color) / <alpha-value>)",
        "secondary-focus": "hsl(var(--secondary-focus-color) / <alpha-value>)",
      },
    },
  },
  plugins: [DaisyUI, Typography],
  daisyui: {
    themes: [],
    base: false,
    logs: false,
  },
} satisfies Config;
