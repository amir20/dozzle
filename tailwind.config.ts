import type { Config } from "tailwindcss";
import Typography from "@tailwindcss/typography";

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
  plugins: [Typography],
} satisfies Config;
