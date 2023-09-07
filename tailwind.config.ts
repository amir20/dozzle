import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";

export default {
  content: ["./assets/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    extend: {
      animation: {
        "bounce-fast": "bounce 1s 2",
      },
    },
  },
  plugins: [DaisyUI],
  daisyui: {
    themes: [
      {
        dozzle: {
          primary: "hsl(171, 100%, 41%)",
          secondary: "hsl(44,  100%, 77%)",
          accent: "#1fb2a6",
          neutral: "#2a323c",
          "base-100": "#dedede",
          "base-content": "#dedede",
        },
      },
    ],
    base: false,
  },
} satisfies Config;
