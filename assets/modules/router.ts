import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import { routes } from "vue-router/auto-routes";
import { setupLayouts } from "virtual:generated-layouts";

export const router = createRouter({
  history: createWebHistory(withBase("/")),
  routes: setupLayouts([...routes]),
});

// After an upgrade the hashed chunks of the old build are gone, so a route this tab has not
// visited yet fails to import. Reloading swaps in the new build instead of showing a dead route.
const isStaleChunkError = (error: unknown) =>
  error instanceof Error && /dynamically imported module|Importing a module script failed/i.test(error.message);

export const install = (app: App) => {
  window.addEventListener("vite:preloadError", () => window.location.reload());

  router.onError((error, to) => {
    if (isStaleChunkError(error)) {
      window.location.href = router.resolve(to).href;
    }
  });

  app.use(router);
};
