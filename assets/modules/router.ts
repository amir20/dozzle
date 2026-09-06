import { type App } from "vue";
import { createRouter, createWebHistory, type Router } from "vue-router";
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

// Every page chunk together is ~50 KB brotli, a quarter of the entry chunk, but importing one
// cold still blocks the navigation long enough to read as lag. Warming them once the app is idle
// makes every later navigation resolve straight from the module cache, on mobile too.
export const prefetchRoutes = (router: Router) => {
  const loaders = router
    .getRoutes()
    .map((route) => route.components?.default)
    .filter((component): component is () => Promise<unknown> => typeof component === "function");

  return Promise.all(loaders.map((load) => load().catch(() => {})));
};

const whenIdle = (callback: () => void) =>
  "requestIdleCallback" in window ? requestIdleCallback(callback, { timeout: 2_000 }) : setTimeout(callback, 500);

export const install = (app: App) => {
  let prefetching = false;
  let navigating = false;

  router.beforeEach(() => {
    navigating = true;
  });
  router.afterEach(() => {
    navigating = false;
  });

  window.addEventListener("vite:preloadError", (event) => {
    // A background prefetch that fails offline must not yank the page out from under someone
    // reading logs. Cancelling swallows the error, which the prefetch ignores anyway; a real
    // navigation still reloads.
    if (prefetching && !navigating) {
      event.preventDefault();
      return;
    }
    window.location.reload();
  });

  router.onError((error, to) => {
    navigating = false;
    if (isStaleChunkError(error)) {
      window.location.href = router.resolve(to).href;
    }
  });

  app.use(router);

  // Save-Data means the browser is asking us not to spend bytes speculatively.
  if (!(navigator as Navigator & { connection?: { saveData?: boolean } }).connection?.saveData) {
    whenIdle(() => {
      prefetching = true;
      prefetchRoutes(router).finally(() => (prefetching = false));
    });
  }
};
