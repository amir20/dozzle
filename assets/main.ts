import "./styles.scss";
import { createApp, App as VueApp } from "vue";
import App from "./App.vue";

const app = createApp(App);
Object.values(import.meta.glob<{ install: (app: VueApp) => void }>("./modules/*.ts", { eager: true })).forEach((i) =>
  i.install?.(app)
);

app.mount("#app");
