import path from "node:path";
import { defineConfig } from "vite";
import Vue from "@vitejs/plugin-vue";
import VueMacros from "unplugin-vue-macros/vite";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import AutoImport from "unplugin-auto-import/vite";
import IconsResolver from "unplugin-icons/resolver";
import VueRouter from "vue-router/vite";
import Layouts from "vite-plugin-vue-layouts";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import svgLoader from "vite-svg-loader";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(() => ({
  base: "./",
  define: {
    // No component uses the options API, so drop that half of the Vue runtime.
    __VUE_OPTIONS_API__: false,
  },
  resolve: {
    alias: {
      "@/": `${path.resolve(import.meta.dirname, "assets")}/`,
      // ansi-to-html drags in the full `entities` HTML entity table (~135 KB raw). We build
      // the converter with escapeXML: false, so only encodeXML is reachable. See the shim.
      entities: path.resolve(import.meta.dirname, "assets/shims/entities.ts"),
    },
  },
  optimizeDeps: {
    // The chat pane is loaded lazily, so vite's startup scan never sees
    // markdown-it (and its five bare deps). It discovers them the first time
    // someone opens the pane, re-optimizes mid-session, and the chunk that
    // triggered it fails with "504 Outdated Optimize Dep". Naming it here means
    // it is pre-bundled before anyone asks.
    include: ["markdown-it"],
  },
  build: {
    manifest: true,
    // App icons are referenced by URL and fetched on demand. Inlining the small ones
    // would drag all ~250 into the entry chunk as base64 for the handful ever shown.
    assetsInlineLimit: (filePath: string) => (filePath.includes("assets/icons/apps/") ? false : undefined),
    rollupOptions: {
      input: "assets/main.ts",
    },
    modulePreload: {
      polyfill: false,
    },
    target: "esnext",
    chunkSizeWarningLimit: 600,
  },
  plugins: [
    VueRouter({
      routesFolder: {
        src: "./assets/pages",
      },
      dts: "./assets/typed-router.d.ts",
      // Async keeps each page out of the entry chunk. Vite resolves the dynamic import
      // against import.meta.url, so this works under a custom base too.
      importMode: "async",
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
      imports: [
        "vue",
        // Replace VueRouterAutoImports with explicit imports:
        {
          "vue-router/auto": ["useRoute", "useRouter", "useLink"],
          "vue-router": ["onBeforeRouteLeave", "onBeforeRouteUpdate"],
        },
        "vue-i18n",
        "pinia",
        "@vueuse/head",
        "@vueuse/core",
      ],
      dts: "assets/auto-imports.d.ts",
      // Recursive: composables live in feature subfolders. A bare dir is scanned
      // one level deep only, which silently drops everything nested.
      dirs: ["assets/composable/**", "assets/stores", "assets/utils/index.ts"],
      vueTemplate: true,
    }),
    VueI18nPlugin({
      runtimeOnly: true,
      strictMessage: false,
      include: [path.resolve(import.meta.dirname, "locales/**")],
    }),
    svgLoader({}),
    tailwindcss(),
  ],
  test: {
    include: ["assets/**/*.spec.ts"],
    setupFiles: ["./assets/test-setup.ts"],
  },
}));
