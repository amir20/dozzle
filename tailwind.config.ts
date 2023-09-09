import type { Config } from "tailwindcss";
import DaisyUI from "daisyui";

export default {
  content: ["./assets/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    extend: {
      animation: {
        "bounce-fast": "bounce 0.5s 2 both",
      },
      colors: {
        "primary-color": "var(--primary-color)",
        scheme: "var(--scheme-main)",
        "scheme-bis": "var(--scheme-main-bis)",
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
          test: "red",
        },
      },
    ],
    base: false,
  },
} satisfies Config;
