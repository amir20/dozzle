import { type App } from "vue";
import { createI18n } from "vue-i18n";
import { locale } from "@/stores/settings";
import type { Locale } from "vue-i18n";

const localesMap = Object.fromEntries(
  Object.entries(import.meta.glob("../../locales/*.yml")).map(([path, loadLocale]) => [
    path.match(/([\w-]*)\.yml$/)?.[1],
    loadLocale,
  ]),
) as Record<Locale, () => Promise<{ default: Record<string, string> }>>;

export const availableLocales = Object.keys(localesMap);

function setI18nLanguage(lang: Locale) {
  i18n.global.locale.value = lang;
  return lang;
}

export const i18n = createI18n({
  legacy: false,
  locale: "",
  fallbackLocale: "en",
  messages: {},
});

const loadedLanguages: string[] = [];
async function loadLanguage(lang: string, setLang = true): Promise<Locale> {
  if (setLang) {
    if (i18n.global.locale.value === lang) return setI18nLanguage(lang);
    if (loadedLanguages.includes(lang)) return setI18nLanguage(lang);
  }

  const messages = await localesMap[lang]();
  i18n.global.setLocaleMessage(lang, messages.default);
  loadedLanguages.push(lang);
  return setI18nLanguage(lang);
}

await loadLanguage("en", false); // load default language

const userLocale = computed(
  () =>
    locale.value ||
    [navigator.language.toLowerCase(), navigator.language.toLowerCase().slice(0, 2)].find((l) =>
      availableLocales.includes(l),
    ) ||
    "en",
);

if (userLocale.value !== "en") {
  await loadLanguage(userLocale.value);
}

watchEffect(() => loadLanguage(userLocale.value));

export const install = (app: App) => app.use(i18n);
export default i18n;
