import path from "path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import IconsResolver from "unplugin-icons/resolver";

export default defineConfig(({ mode }) => ({
  resolve: {
    alias: {
      "@/": `${path.resolve(__dirname, "assets")}/`,
    },
  },
  base: mode === "production" ? "/<__BASE__>" : "/",
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
    htmlPlugin(mode),
  ],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080/",
      },
    },
  },
}));

const htmlPlugin = (mode) => {
  return {
    name: "html-transform",
    transformIndexHtml(html) {
      return mode === "production" ? html.replaceAll("/<__BASE__>", "{{ .Base }}") : html;
    },
  };
};
