import Vue from "vue";
import VueRouter from "vue-router";
import App from "./App.vue";
import Index from "./pages/Index.vue";
import Container from "./pages/Container.vue";

Vue.use(VueRouter);

const routes = [
  { path: "/", component: Index },
  {
    path: "/container/:id",
    component: Container,
    name: "container",
    props: true
  }
];

const router = new VueRouter({
  mode: "history",
  routes
});

new Vue({
  router,
  render: h => h(App)
}).$mount("#app");
