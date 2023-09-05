import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";

export default {
  content: ["./assets/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [DaisyUI],
  daisyui: {
    themes: [],
    base: false,
  },
} satisfies Config;
