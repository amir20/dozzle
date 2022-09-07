import { type App } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import { Container, Settings, Index, Show, ContainerNotFound, PageNotFound, Login } from "../pages";
import config from "@/stores/config";

const routes = [
  {
    path: "/",
    component: Index,
    name: "default",
  },
  {
    path: "/container/:id",
    component: Container,
    name: "container",
    props: true,
  },
  {
    path: "/container/:pathMatch(.*)",
    component: ContainerNotFound,
    name: "container-not-found",
  },
  {
    path: "/settings",
    component: Settings,
    name: "settings",
  },
  {
    path: "/show",
    component: Show,
    name: "show",
  },
  {
    path: "/login",
    component: Login,
    name: "login",
  },
  {
    path: "/:pathMatch(.*)*",
    component: PageNotFound,
    name: "page-not-found",
  },
];

export const install = (app: App) => {
  const router = createRouter({
    history: createWebHistory(`${config.base}/`),
    routes,
  });

  app.use(router);
};
