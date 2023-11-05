import { toRefs } from "@vueuse/core";
const DOZZLE_SETTINGS_KEY = "DOZZLE_SETTINGS";

export type Settings = {
  search: boolean;
  size: "small" | "medium" | "large";
  menuWidth: number;
  smallerScrollbars: boolean;
  showTimestamp: boolean;
  showStd: boolean;
  showAllContainers: boolean;
  lightTheme: "auto" | "dark" | "light";
  hourStyle: "auto" | "24" | "12";
  softWrap: boolean;
  collapseNav: boolean;
  automaticRedirect: boolean;
};
export const DEFAULT_SETTINGS: Settings = {
  search: true,
  size: "medium",
  menuWidth: 15,
  smallerScrollbars: false,
  showTimestamp: true,
  showStd: false,
  showAllContainers: false,
  lightTheme: "auto",
  hourStyle: "auto",
  softWrap: true,
  collapseNav: false,
  automaticRedirect: true,
};

export const settings = useStorage(DOZZLE_SETTINGS_KEY, DEFAULT_SETTINGS);
settings.value = { ...DEFAULT_SETTINGS, ...settings.value, ...config.serverSettings };

if (config.user) {
  watch(settings, (value) => {
    fetch(withBase("/api/profile/settings"), {
      method: "PUT",
      body: JSON.stringify(value),
    });
  });
}

export const {
  collapseNav,
  softWrap,
  hourStyle,
  lightTheme,
  showAllContainers,
  showTimestamp,
  showStd,
  smallerScrollbars,
  menuWidth,
  size,
  search,
  automaticRedirect,
} = toRefs(settings);
