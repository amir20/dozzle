import type { Config } from "tailwindcss";

export default {
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
    },
  },
} satisfies Config;
