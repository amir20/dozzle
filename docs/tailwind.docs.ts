import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";

export default {
  future: {
    hoverOnlyWhenSupported: true,
  },
  darkMode: "selector",
  content: ["docs/.vitepress/theme/**/*.{vue,js,ts}"],
  plugins: [DaisyUI],
  daisyui: {
    themes: [],
    base: false,
    logs: false,
  },
} satisfies Config;
