import Vue from "vue";
import VueRouter from "vue-router";
import Meta from "vue-meta";
import Switch from "buefy/dist/esm/switch";
import Radio from "buefy/dist/esm/radio";
import Field from "buefy/dist/esm/field";
import store from "./store";
import config from "./store/config";
import App from "./App.vue";
import { Container, Settings, Index, Show, ContainerNotFound, PageNotFound } from "./pages";

Vue.use(VueRouter);
Vue.use(Meta);
Vue.use(Switch);
Vue.use(Radio);
Vue.use(Field);

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
    path: "/*",
    component: PageNotFound,
    name: "page-not-found",
  },
];

const router = new VueRouter({
  mode: "history",
  base: config.base + "/",
  routes,
});

new Vue({
  router,
  store,
  render: (h) => h(App),
}).$mount("#app");
