import { toRefs } from "@vueuse/core";
const DOZZLE_SETTINGS_KEY = "DOZZLE_SETTINGS";

export const DEFAULT_SETTINGS: {
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
} = {
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
};

export const settings = useStorage(DOZZLE_SETTINGS_KEY, DEFAULT_SETTINGS);
settings.value = { ...DEFAULT_SETTINGS, ...settings.value };

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
} = toRefs(settings);
