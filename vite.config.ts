import path from "path";
import { defineConfig } from "vite";
import IconsResolver from "unplugin-icons/resolver";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";

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
