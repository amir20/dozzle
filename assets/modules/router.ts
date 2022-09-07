import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import routes from "~pages";
import config from "@/stores/config";



export const install = (app: App) => {
  const router = createRouter({
    history: createWebHistory(`${config.base}/`),
    routes,
  });

  app.use(router);
};
