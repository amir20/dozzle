import path from "path";
import { defineConfig } from "vite";
import Vue from "@vitejs/plugin-vue";
import VueMacros from "unplugin-vue-macros/vite";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import AutoImport from "unplugin-auto-import/vite";
import IconsResolver from "unplugin-icons/resolver";
import Pages from "vite-plugin-pages";
import Layouts from "vite-plugin-vue-layouts";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import { compression } from "vite-plugin-compression2";

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
  },
  plugins: [
    VueMacros({
      plugins: {
        vue: Vue(),
      },
    }),
    Icons({
      autoInstall: true,
    }),
    Pages({
      dirs: "assets/pages",
      importMode: "sync",
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
      imports: ["vue", "vue-router", "vue-i18n", "vue/macros", "pinia", "@vueuse/head", "@vueuse/core"],
      dts: "assets/auto-imports.d.ts",
      dirs: ["assets/composable", "assets/stores", "assets/utils"],
      vueTemplate: true,
    }),
    VueI18nPlugin({
      runtimeOnly: true,
      compositionOnly: true,
      strictMessage: false,
      include: [path.resolve(__dirname, "locales/**")],
    }),
    compression({ algorithm: "brotliCompress", exclude: [/\.(html)$/] }),
  ],
  server: {
    watch: {
      ignored: ["**/data/**"],
    },
    proxy: {
      "/api": {
        target: {
          host: "127.0.0.1",
          port: 3100,
        },
        changeOrigin: false,
      },
    },
  },
  test: {
    include: ["assets/**/*.spec.ts"],
  },
}));
