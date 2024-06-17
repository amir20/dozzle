import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import { routes } from "vue-router/auto-routes";
import { setupLayouts } from "virtual:generated-layouts";

export const router = createRouter({
  history: createWebHistory(withBase("/")),
  routes: setupLayouts(routes),
});

export const install = (app: App) => {
  app.use(router);
};
