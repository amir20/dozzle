import path from "node:path";
import { defineConfig } from "vite";
import Vue from "@vitejs/plugin-vue";
import VueMacros from "unplugin-vue-macros/vite";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import AutoImport from "unplugin-auto-import/vite";
import IconsResolver from "unplugin-icons/resolver";
import VueRouter from "unplugin-vue-router/vite";
import Layouts from "vite-plugin-vue-layouts";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import { VueRouterAutoImports } from "unplugin-vue-router";
import svgLoader from "vite-svg-loader";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(() => ({
  resolve: {
    alias: {
      "@/": `${path.resolve(__dirname, "assets")}/`,
    },
  },
  build: {
    manifest: true,
    rollupOptions: {
      input: "assets/main.ts",
    },
    modulePreload: {
      polyfill: false,
    },
    target: "esnext",
  },
  plugins: [
    VueRouter({
      routesFolder: {
        src: "./assets/pages",
      },
      dts: "./assets/typed-router.d.ts",
      importMode: "sync",
    }),
    VueMacros({
      plugins: {
        vue: Vue(),
      },
    }),
    Icons({
      autoInstall: true,
    }),

    Layouts({
      layoutsDirs: "assets/layouts",
    }),
    Components({
      dirs: ["assets/components"],
      resolvers: [
        IconsResolver({
          componentPrefix: "",
        }),
      ],

      dts: "assets/components.d.ts",
    }),
    AutoImport({
      imports: ["vue", VueRouterAutoImports, "vue-i18n", "pinia", "@vueuse/head", "@vueuse/core"],
      dts: "assets/auto-imports.d.ts",
      dirs: ["assets/composable", "assets/stores", "assets/utils"],
      vueTemplate: true,
    }),
    VueI18nPlugin({
      runtimeOnly: true,
      strictMessage: false,
      include: [path.resolve(__dirname, "locales/**")],
    }),
    svgLoader({}),
    tailwindcss(),
  ],
  test: {
    include: ["assets/**/*.spec.ts"],
  },
}));
