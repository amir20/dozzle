import type { Config } from "tailwindcss";
import Typography from "@tailwindcss/typography";

export default {
  future: {
    hoverOnlyWhenSupported: true,
  },
  content: ["./assets/**/*.{vue,js,ts}", "./public/index.html"],
  theme: {
    extend: {
      blur: {
        xs: "1px",
      },
      animation: {
        "bounce-fast": "bounce 0.5s 2 both",
      },
      backgroundImage: {
        "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
      },
      colors: {
        green: "oklch(69% 0.119722 188.479048)",
        red: "oklch(64% 0.218 28.85)",
        purple: "oklch(51.49% 0.215 321.03)",
        blue: "oklch(65% 0.171 249.5)",
        orange: "oklch(70% 0.186 48.13)",
      },
    },
  },
  plugins: [Typography],
} satisfies Config;
