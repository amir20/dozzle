import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";
import Typography from "@tailwindcss/typography";
import Container from "@tailwindcss/container-queries";

export default {
  future: {
    hoverOnlyWhenSupported: true,
  },
  content: ["./assets/**/*.{vue,js,ts}", "./public/index.html"],
  theme: {
    extend: {
      animation: {
        "bounce-fast": "bounce 0.5s 2 both",
      },
      colors: {
        green: "oklch(69% 0.119722 188.479048)",
        red: "oklch(64% 0.218 28.85)",
        purple: "oklch(51.49% 0.215 321.03)",
        blue: "oklch(65% 0.171 249.5)",
        orange: "oklch(70% 0.186 48.13)",
        base: "oklch(var(--base-color) / <alpha-value>)",
        "base-darker": "oklch(var(--base-darker-color) / <alpha-value>)",
        "base-lighter": "oklch(var(--base-lighter-color) / <alpha-value>)",
        "base-content": "oklch(var(--base-content-color) / <alpha-value>)",
        primary: "oklch(var(--primary-color) / <alpha-value>)",
        secondary: "oklch(var(--secondary-color) / <alpha-value>)",
      },
    },
  },
  plugins: [DaisyUI, Typography, Container],
  daisyui: {
    themes: [],
    base: false,
    logs: false,
  },
} satisfies Config;
