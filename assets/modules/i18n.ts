import { type App } from "vue";
import { createI18n } from "vue-i18n";
import { locale } from "@/stores/settings";

import messages from "@intlify/unplugin-vue-i18n/messages";

const defaultLocale = messages?.hasOwnProperty(navigator.language)
  ? navigator.language
  : navigator.language.slice(0, 2);

const i18n = createI18n({
  legacy: false,
  locale: locale.value || defaultLocale,
  fallbackLocale: "en",
  messages,
});

watch(locale, (value) => {
  i18n.global.locale.value = value || defaultLocale;
});

export const install = (app: App) => app.use(i18n);
export default i18n;
