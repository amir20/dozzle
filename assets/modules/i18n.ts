import { type App } from "vue";
import { createI18n } from "vue-i18n";
import { locale } from "@/stores/settings";
import type { Locale } from "vue-i18n";
import en from "../../locales/en.yml";

// `en` is the fallback locale and is always needed before the app can mount, so it is
// imported statically and rides along in the entry chunk. Awaiting it as a dynamic
// import cost an extra round trip before the first render. Every other locale stays
// lazy, so `en` is excluded from the glob to avoid emitting a chunk nothing fetches.
const localesMap = Object.fromEntries(
  Object.entries(import.meta.glob(["../../locales/*.yml", "!../../locales/en.yml"])).map(([path, loadLocale]) => [
    path.match(/([\w-]*)\.yml$/)?.[1],
    loadLocale,
  ]),
) as Record<Locale, () => Promise<{ default: Record<string, string> }>>;

export const availableLocales = ["en", ...Object.keys(localesMap)].sort();

function setI18nLanguage(lang: Locale) {
  i18n.global.locale.value = lang;
  return lang;
}

export const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackLocale: "en",
  // Widened so `locale` stays a plain string; a bare literal narrows it to "en".
  messages: { en } as Record<Locale, typeof en>,
});

const loadedLanguages: string[] = ["en"];
async function loadLanguage(lang: string): Promise<Locale> {
  if (i18n.global.locale.value === lang) return setI18nLanguage(lang);
  if (loadedLanguages.includes(lang)) return setI18nLanguage(lang);

  const messages = await localesMap[lang]();
  i18n.global.setLocaleMessage(lang, messages.default);
  loadedLanguages.push(lang);
  return setI18nLanguage(lang);
}

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
