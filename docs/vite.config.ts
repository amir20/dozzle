import path from "path";
import { defineConfig } from "vite";

import Components from "unplugin-vue-components/vite";
import webfontDownload from "vite-plugin-webfont-dl";

export default defineConfig({
  plugins: [
    Components({
      dirs: [path.resolve(__dirname, ".vitepress/theme/components")],
      extensions: ["vue", "md"],
      include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
      dts: true,
    }),
    webfontDownload([
      "https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,400..900;1,400..900&display=swap",
    ]),
  ],
});
