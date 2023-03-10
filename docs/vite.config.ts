import path from "path";
import { defineConfig } from "vite";

import Components from "unplugin-vue-components/vite";
import Unocss from "unocss/vite";

export default defineConfig({
  plugins: [
    Components({
      dirs: [path.resolve(__dirname, ".vitepress/theme/components")],
      extensions: ["vue", "md"],
      include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
      dts: true,
    }),

    Unocss(),
  ],
});
