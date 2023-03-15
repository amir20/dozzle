import { type App } from "vue";
import { createPinia } from "pinia";

export const install = (app: App) => {
  const pinia = createPinia();
  app.use(pinia);
};
