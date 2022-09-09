import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import pages from "~pages";
import { setupLayouts } from "virtual:generated-layouts";
import config from "@/stores/config";

export const install = (app: App) => {
  const routes = setupLayouts(pages);

  const router = createRouter({
    history: createWebHistory(`${config.base}/`),
    routes,
  });

  app.use(router);
};
