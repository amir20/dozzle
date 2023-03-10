import { defineConfig, presetAttributify, presetIcons, presetUno, presetWebFonts } from "unocss";

import { presetTypography } from "@unocss/preset-typography";
import transformerDirectives from "@unocss/transformer-directives";

export default defineConfig({
  shortcuts: [
    [
      "btn",
      "px-4 py-1 rounded inline-block bg-teal-600 text-white cursor-pointer hover:bg-teal-700 disabled:cursor-default disabled:bg-gray-600 disabled:opacity-50",
    ],
    [
      "icon-btn",
      "text-[0.9em] inline-block cursor-pointer select-none opacity-75 transition duration-200 ease-in-out hover:opacity-100 hover:text-teal-600 !outline-none",
    ],
    ["main-bg", "bg-main-light dark:bg-main-dark"],
    [
      "btn-primary",
      "rounded-full no-underline py-3 px-4 bg-primary hover:bg-primary-dark text-white hover:text-white text-lg focus:outline-none border-primary-light border-1 border-solid",
    ],
    [/^circle-(\w+)$/, ([, c]) => `rounded-full bg-${c}500 w-3 h-3`],
  ],
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
