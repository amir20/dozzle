import { type App as VueApp, createApp } from "vue";
import App from "./App.vue";
import "./main.css";

const app = createApp(App);
const modules = import.meta.glob<{ install: (app: VueApp) => void }>("./modules/*.ts", { eager: true });
for (const path in modules) {
  modules[path].install(app);
}
app.mount("#app");
