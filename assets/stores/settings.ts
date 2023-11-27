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

export const settings = useProfileStorage("settings", DEFAULT_SETTINGS);

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
