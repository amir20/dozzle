import { defineConfig, presetAttributify, presetIcons, presetUno, presetWebFonts } from "unocss";

import { presetTypography } from "@unocss/preset-typography";
import transformerDirectives from "@unocss/transformer-directives";

export default defineConfig({
  shortcuts: [[/^circle-(\w+)$/, ([, c]) => `rounded-full bg-${c}500 w-3 h-3`]],
  transformers: [transformerDirectives()],
  presets: [
    presetUno(),
    presetAttributify(),
    presetTypography(),
    presetIcons({
      scale: 1.2,
      warn: true,
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
