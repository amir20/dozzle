import "./main.css";
import { createApp, App as VueApp } from "vue";
import App from "./App.vue";

const app = createApp(App);
// Specs live next to the modules they cover, and an eager glob would drag vitest into the entry chunk.
Object.values(
  import.meta.glob<{ install: (app: VueApp) => void }>(["./modules/*.ts", "!./modules/*.spec.ts"], {
    eager: true,
  }),
).forEach((i) => i.install?.(app));
app.mount("#app");

if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register(withBase("/sw.js"));
}
