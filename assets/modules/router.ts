import { setupLayouts } from "virtual:generated-layouts";
import type { App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import pages from "~pages";

const routes = setupLayouts(pages);
export const router = createRouter({
  history: createWebHistory(withBase("/")),
  routes,
});

export const install = (app: App) => {
  app.use(router);
};
