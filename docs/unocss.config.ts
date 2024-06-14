import { defineConfig, presetIcons, presetUno, presetWebFonts } from "unocss";

import { presetTypography } from "@unocss/preset-typography";
import transformerDirectives from "@unocss/transformer-directives";

export default defineConfig({
  shortcuts: [[/^circle-(\w+)$/, ([, c]) => `rounded-full bg-${c}500 w-2 h-2 lg:w-3 lg:h-3`]],
  transformers: [transformerDirectives()],
  presets: [
    presetUno(),
    presetTypography(),
    presetIcons({
      scale: 1.2,
      warn: true,
      autoInstall: true,
    }),
    presetWebFonts({
      fonts: {
        sans: "Roboto:200",
        playfair: [
          {
            name: "Playfair Display",
            weights: [100, 200, 400, 700],
          },
        ],
      },
    }),
  ],
  theme: {
    colors: {},
  },
});
