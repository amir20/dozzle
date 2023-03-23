import path from "path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import AutoImport from "unplugin-auto-import/vite";
import IconsResolver from "unplugin-icons/resolver";
import Pages from "vite-plugin-pages";
import Layouts from "vite-plugin-vue-layouts";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";

export default defineConfig(() => ({
  resolve: {
    alias: {
      "@/": `${path.resolve(__dirname, "assets")}/`,
    },
  },
  experimental: {
    renderBuiltUrl(filename: string, { type }: { type: "public" | "asset" }) {
      if (type === "asset") {
        return `{{ .Base }}/${filename}`;
      }
      return filename;
    },
  },
  plugins: [
    vue({
      reactivityTransform: true,
      template: {
        compilerOptions: {
          whitespace: "preserve",
        },
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
      dirs: ["assets/composables", "assets/stores", "assets/utils"],
      vueTemplate: true,
    }),
    VueI18nPlugin({
      runtimeOnly: true,
      compositionOnly: true,
      strictMessage: false,
      include: [path.resolve(__dirname, "locales/**")],
    }),
  ],
  server: {
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
}));
