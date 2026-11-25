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
  // Was collapseCloudRail. Renamed when the rail became the app's nav, so a strip
  // someone hid when it only held cloud panels does not now hide settings and logout.
  collapseRail: boolean;
  automaticRedirect: "instant" | "delayed" | "none";
  locale: string;
  groupContainers: "always" | "at-least-2" | "never";
  showImageUpdateAlert: boolean;
  showAppIcons: boolean;
  highlightErrors: boolean;
  // "raw" keeps each line's own field order; "readable" leads a structured line with
  // its message (and, for a request, method, path, status and duration).
  lineStyle: "raw" | "readable";
  // Lines in a row that say the same thing show once, with how many times.
  foldRepeats: boolean;
  terminalFontSize: number;
  // dashboard table columns the user switched off; the name column cannot be
  hiddenColumns: string[];
  movingAverage: boolean;
};
export const DEFAULT_SETTINGS: Settings = {
  search: true,
  compact: false,
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
  collapseRail: false,
  automaticRedirect: "delayed",
  locale: "",
  groupContainers: "at-least-2",
  showImageUpdateAlert: false,
  showAppIcons: true,
  highlightErrors: true,
  lineStyle: "raw",
  foldRepeats: false,
  terminalFontSize: 13,
  // Stored as what is hidden, so a column added later shows up unless it is listed here.
  hiddenColumns: ["uptime", "diskio"],
  movingAverage: true,
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
  collapseRail,
  compact,
  softWrap,
  hourStyle,
  dateLocale,
  lightTheme,
  showAllContainers,
  showTimestamp,
  showStd,
  smallerScrollbars,
  showImageUpdateAlert,
  menuWidth,
  size,
  search,
  locale,
  automaticRedirect,
  groupContainers,
  showAppIcons,
  highlightErrors,
  lineStyle,
  foldRepeats,
  terminalFontSize,
  hiddenColumns,
  movingAverage,
} = toRefs<Settings>(settings);
