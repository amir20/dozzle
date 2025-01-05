import path from "path";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

import Components from "unplugin-vue-components/vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    Components({
      dirs: [path.resolve(__dirname, ".vitepress/theme/components")],
      extensions: ["vue", "md"],
      include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
      dts: true,
    }),
  ],
});
