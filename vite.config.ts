import path from "path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import IconsResolver from "unplugin-icons/resolver";
import Pages from "vite-plugin-pages";

export default defineConfig(({ mode }) => ({
  resolve: {
    alias: {
      "@/": `${path.resolve(__dirname, "assets")}/`,
    },
  },
  base: mode === "production" ? "/{{ .Base }}/" : "/",
  plugins: [
    vue(
      {
        reactivityTransform: true,
      }
    ),
    Icons({
      autoInstall: true,
    }),
    Pages({
      dirs: "assets/pages",
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
        target: "http://localhost:3100/",
      },
    },
  },
}));

const htmlPlugin = (mode) => {
  return {
    name: "html-transform",
    enforce: "post" as const,
    transformIndexHtml(html) {
      return mode === "production" ? html.replaceAll("/{{ .Base }}/", "{{ .Base }}/") : html;
    },
  };
};
