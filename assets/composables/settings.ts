import { useStorage } from "@vueuse/core";
import { computed } from "vue";

export const DOZZLE_SETTINGS_KEY = "DOZZLE_SETTINGS";

export const DEFAULT_SETTINGS: {
  search: boolean;
  size: "small" | "medium" | "large";
  menuWidth: number;
  smallerScrollbars: boolean;
  showTimestamp: boolean;
  showAllContainers: boolean;
  lightTheme: "auto" | "dark" | "light";
  hourStyle: "auto" | "24" | "12";
  softWrap: boolean;
} = {
  search: true,
  size: "medium",
  menuWidth: 15,
  smallerScrollbars: false,
  showTimestamp: true,
  showAllContainers: false,
  lightTheme: "auto",
  hourStyle: "auto",
  softWrap: true,
};

export const settings = useStorage(DOZZLE_SETTINGS_KEY, DEFAULT_SETTINGS);
settings.value = {...DEFAULT_SETTINGS, ...settings.value};

export const search = computed({
  get: () => settings.value.search,
  set: (value) => (settings.value.search = value),
});

export const size = computed({
  get: () => settings.value.size,
  set: (value) => (settings.value.size = value),
});

export const menuWidth = computed({
  get: () => settings.value.menuWidth,
  set: (value) => (settings.value.menuWidth = value),
});
export const smallerScrollbars = computed({
  get: () => settings.value.smallerScrollbars,
  set: (value) => (settings.value.smallerScrollbars = value),
});

export const showTimestamp = computed({
  get: () => settings.value.showTimestamp,
  set: (value) => (settings.value.showTimestamp = value),
});

export const showAllContainers = computed({
  get: () => settings.value.showAllContainers,
  set: (value) => (settings.value.showAllContainers = value),
});

export const lightTheme = computed({
  get: () => settings.value.lightTheme,
  set: (value) => (settings.value.lightTheme = value),
});

export const hourStyle = computed({
  get: () => settings.value.hourStyle,
  set: (value) => (settings.value.hourStyle = value),
});

export const softWrap = computed({
  get: () => settings.value.softWrap,
  set: (value) => (settings.value.softWrap = value),
});
