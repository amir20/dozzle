import { createApp } from "vue";
import App from "./App.vue";
import "./main.css";

const app = createApp(App);

for (const module of import.meta.globEager("./modules/*.ts")) {
  module[1].install?.(app);
}

app.mount("#app");
