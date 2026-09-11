// @ts-check
import path from "node:path";
import eslintPluginTailwindcss from "eslint-plugin-tailwindcss";
import { defineConfig, globalIgnores } from "eslint/config";
import vueParser from "vue-eslint-parser";
import tsParser from "@typescript-eslint/parser";

// This repo does not lint JS/TS style with ESLint. Prettier owns formatting and
// class order (prettier-plugin-tailwindcss), so nothing stylistic belongs here.
// The only job is catching Tailwind classes the compiler says can be written
// better, which the IDE extension shows but CI never did.
export default defineConfig([
  globalIgnores([
    "dist/**",
    "docs/.vitepress/cache/**",
    "docs/.vitepress/dist/**",
    "assets/auto-imports.d.ts",
    "assets/components.d.ts",
    "assets/typed-router.d.ts",
    "assets/types/graphql.ts",
  ]),
  {
    files: ["**/*.{js,mjs,ts,mts,vue}"],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tsParser,
        ecmaVersion: "latest",
        sourceType: "module",
        extraFileExtensions: [".vue"],
      },
    },
    plugins: { tailwindcss: eslintPluginTailwindcss },
    // Only Tailwind rules run here, so an `eslint-disable` in a generated file
    // is aimed at somebody else's linter, not ours.
    linterOptions: { reportUnusedDisableDirectives: "off" },
    settings: {
      tailwindcss: /** @type {import('eslint-plugin-tailwindcss').PluginSettings} */ ({
        // Absolute. A relative path here is resolved against the file being
        // linted, not the config, so it misses for anything outside assets/.
        cssConfigPath: path.join(import.meta.dirname, "assets/main.css"),
      }),
    },
    // The recommended config turns on more than this repo wants. Every rule the
    // plugin ships is listed so a future reader sees a decision, not a gap.
    rules: {
      // The point of the exercise: `min-w-[4px]` when `min-w-1` exists.
      "tailwindcss/no-unnecessary-arbitrary-value": "error",
      "tailwindcss/enforces-shorthand": "error",
      "tailwindcss/no-contradicting-classname": "error",
      "tailwindcss/enforces-negative-arbitrary-values": "error",
      // v3 leftovers: `!border-warning` instead of `border-warning!`.
      "tailwindcss/important-modifier-suffix": "error",
      // Rewrites plenty of classes that are already correct and readable
      // (`end-1` to `inset-e-1`, `break-words` to `wrap-break-word`). It is a
      // canonicalizer, not a bug finder.
      "tailwindcss/enforces-canonical-classname": "off",
      // prettier-plugin-tailwindcss already sorts classes on every commit, and
      // the two orderings are not identical. Leaving this on makes the two
      // tools rewrite each other.
      "tailwindcss/classnames-order": "off",
      // daisyUI, and our own component classes in main.css, are custom as far
      // as the compiler is concerned. 333 reports, none of them a bug.
      "tailwindcss/no-custom-classname": "off",
      // Arbitrary values are fine here. Most in the tree are em based or
      // off-scale with no Tailwind equivalent; only the unnecessary ones the
      // rule above catches are a problem.
      "tailwindcss/no-arbitrary-value": "off",
    },
  },
]);
