import { toRefs } from "@vueuse/core";

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
  dateLocale: "auto" | "en-US" | "en-GB" | "de-DE" | "en-CA";
  softWrap: boolean;
  collapseNav: boolean;
  automaticRedirect: boolean;
  locale: string | undefined;
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
  dateLocale: "auto",
  softWrap: true,
  collapseNav: false,
  automaticRedirect: true,
  locale: undefined,
};

export const settings = useProfileStorage("settings", DEFAULT_SETTINGS);

export const {
  collapseNav,
  softWrap,
  hourStyle,
  dateLocale,
  lightTheme,
  showAllContainers,
  showTimestamp,
  showStd,
  smallerScrollbars,
  menuWidth,
  size,
  search,
  locale,
  automaticRedirect,
} = toRefs(settings);
