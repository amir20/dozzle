import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import pages from "~pages";
import { setupLayouts } from "virtual:generated-layouts";

const routes = setupLayouts(pages);
export const router = createRouter({
  history: createWebHistory(withBase("/")),
  routes,
});

export const install = (app: App) => {
  app.use(router);
};
