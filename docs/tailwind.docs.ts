import type { Config } from "tailwindcss";

export default {
  future: {
    hoverOnlyWhenSupported: true,
  },
  darkMode: "selector",
  content: ["docs/.vitepress/theme/**/*.{vue,js,ts}"],
} satisfies Config;
