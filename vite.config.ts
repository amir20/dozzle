import path from "path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import IconsResolver from "unplugin-icons/resolver";
import AutoImport from "unplugin-auto-import/vite";

export default defineConfig({
  resolve: {
    alias: {
      "@/": `${path.resolve(__dirname, "assets")}/`,
    },
  },
  plugins: [
    vue(),
    Icons({
      autoInstall: true,
    }),
    AutoImport({
      imports: [
        "vue",
        "vue-router",
        // 'vue-i18n',
        // '@vueuse/head',
        // '@vueuse/core',
      ],
      dts: "assets/auto-imports.d.ts",
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
  ],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080/",
      },
    },
  },
});
