import { type App } from "vue";
import { createI18n } from "vue-i18n";

import messages from "@intlify/unplugin-vue-i18n/messages";

const locale = messages?.hasOwnProperty(navigator.language) ? navigator.language : navigator.language.slice(0, 2);

const i18n = createI18n({
  legacy: false,
  locale: locale,
  fallbackLocale: "en",
  messages,
});

export const install = (app: App) => {
  app.use(i18n);
};

export default i18n;
