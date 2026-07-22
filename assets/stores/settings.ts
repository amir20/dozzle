import { computed } from "vue";
import { toRefs } from "@vueuse/core";

export type Settings = {
  search: boolean;
  size: "small" | "medium" | "large";
  compact: boolean;
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
  automaticRedirect: "instant" | "delayed" | "none";
  locale: string;
  groupContainers: "always" | "at-least-2" | "never";
  cpuDisplayMode: "utilization" | "cores";
  settingsAsPopup: boolean;
};
// Shared sidebar sizing (percent of the window width) used by the layout and
// the "reset sidebar width" action so they always agree on one value.
export const DEFAULT_MENU_WIDTH = 15;
export const MIN_MENU_WIDTH = 10;

export const DEFAULT_SETTINGS: Settings = {
  search: true,
  compact: false,
  size: "medium",
  menuWidth: DEFAULT_MENU_WIDTH,
  smallerScrollbars: false,
  showTimestamp: true,
  showStd: false,
  showAllContainers: false,
  lightTheme: "auto",
  hourStyle: "auto",
  dateLocale: "auto",
  softWrap: true,
  collapseNav: false,
  automaticRedirect: "delayed",
  locale: "",
  groupContainers: "at-least-2",
  cpuDisplayMode: "utilization",
  settingsAsPopup: false,
};

export const settings = useProfileStorage("settings", DEFAULT_SETTINGS);

// @ts-ignore: automaticRedirect is now a string enum, but might be a boolean in older data
if (settings.value.automaticRedirect === true) {
  settings.value.automaticRedirect = "delayed";
  // @ts-ignore: automaticRedirect is now a string enum, but might be a boolean in older data
} else if (settings.value.automaticRedirect === false) {
  settings.value.automaticRedirect = "none";
}

export const {
  collapseNav,
  compact,
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
  groupContainers,
  cpuDisplayMode,
  settingsAsPopup,
} = toRefs(settings.value);

// Reset the sidebar to its default width. Lives here because it operates purely
// on this store's state, and is shared by the toolbars' "reset sidebar width"
// action. Disabled while the sidebar is collapsed or already at the default.
export const canResetMenuWidth = computed(
  () => !collapseNav.value && Math.abs(menuWidth.value - DEFAULT_MENU_WIDTH) > 0.01,
);

export function resetMenuWidth() {
  if (canResetMenuWidth.value) menuWidth.value = DEFAULT_MENU_WIDTH;
}
