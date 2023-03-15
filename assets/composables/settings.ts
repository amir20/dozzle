const DOZZLE_SETTINGS_KEY = "DOZZLE_SETTINGS";

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
  collapseNav: boolean;
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
  collapseNav: false,
};

const settings = useStorage(DOZZLE_SETTINGS_KEY, DEFAULT_SETTINGS);
settings.value = { ...DEFAULT_SETTINGS, ...settings.value };

const search = computed({
  get: () => settings.value.search,
  set: (value) => (settings.value.search = value),
});

const size = computed({
  get: () => settings.value.size,
  set: (value) => (settings.value.size = value),
});

const menuWidth = computed({
  get: () => settings.value.menuWidth,
  set: (value) => (settings.value.menuWidth = value),
});
const smallerScrollbars = computed({
  get: () => settings.value.smallerScrollbars,
  set: (value) => (settings.value.smallerScrollbars = value),
});

const showTimestamp = computed({
  get: () => settings.value.showTimestamp,
  set: (value) => (settings.value.showTimestamp = value),
});

const showAllContainers = computed({
  get: () => settings.value.showAllContainers,
  set: (value) => (settings.value.showAllContainers = value),
});

const lightTheme = computed({
  get: () => settings.value.lightTheme,
  set: (value) => (settings.value.lightTheme = value),
});

const hourStyle = computed({
  get: () => settings.value.hourStyle,
  set: (value) => (settings.value.hourStyle = value),
});

const softWrap = computed({
  get: () => settings.value.softWrap,
  set: (value) => (settings.value.softWrap = value),
});

const collapseNav = computed({
  get: () => settings.value.collapseNav,
  set: (value) => (settings.value.collapseNav = value),
});

export {
  collapseNav,
  softWrap,
  hourStyle,
  lightTheme,
  showAllContainers,
  showTimestamp,
  smallerScrollbars,
  menuWidth,
  size,
  search,
  settings,
};
