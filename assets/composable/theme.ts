import { lightTheme } from "@/stores/settings";

// Deliberately module scope, not per caller. useColorMode() is expensive to
// instantiate: it registers a matchMedia listener and a localStorage watcher, and its
// immediate watch injects a `* { transition: none }` stylesheet into <head>, mutates
// html.classList, and reads back a computed style to force a synchronous whole-document
// recalc. Instantiating it per component made every ContainerIcon in a table row cost
// two full reflows, so rendering N containers cost 2N of them.
const mode = useColorMode();

const theme = computed(() => (lightTheme.value === "auto" ? mode.value : lightTheme.value));
const isDark = computed(() => theme.value === "dark");

/**
 * Resolves the tri-state `lightTheme` preference down to the theme actually painted,
 * mirroring what App.vue writes to `data-theme`. Needed anywhere the choice of asset,
 * rather than the choice of CSS, depends on the theme.
 */
export function useResolvedTheme() {
  return { theme, isDark };
}
