import "./styles.scss";

import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import Meta from "vue-meta";
import Switch from "buefy/dist/esm/switch";
import Radio from "buefy/dist/esm/radio";
import Field from "buefy/dist/esm/field";
import Modal from "buefy/dist/esm/modal";
import Tooltip from "buefy/dist/esm/tooltip";
import Autocomplete from "buefy/dist/esm/autocomplete";

import store from "./store";
import config from "./store/config";
import App from "./App.vue";
import { Container, Settings, Index, Show, ContainerNotFound, PageNotFound, Login } from "./pages";

// Vue.use(Meta);
// Vue.use(Switch);
// Vue.use(Radio);
// Vue.use(Field);
// Vue.use(Modal);
// Vue.use(Tooltip);
// Vue.use(Autocomplete);

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
    path: "/container/*",
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
    path: "/*",
    component: PageNotFound,
    name: "page-not-found",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

const app = createApp(App);
app.use(router);
app.use(store);
app.mount("#app");
