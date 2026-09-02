import { lightTheme } from "@/stores/settings";

/**
 * Resolves the tri-state `lightTheme` preference down to the theme actually painted,
 * mirroring what App.vue writes to `data-theme`. Needed anywhere the choice of asset,
 * rather than the choice of CSS, depends on the theme.
 */
export function useResolvedTheme() {
  const mode = useColorMode();
  const theme = computed(() => (lightTheme.value === "auto" ? mode.value : lightTheme.value));

  return { theme, isDark: computed(() => theme.value === "dark") };
}
