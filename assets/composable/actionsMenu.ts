// Shared open/close behavior for the container action dropdowns. daisyUI's
// hover dropdown suppresses itself on tap-focus (breaking touch) and leaves
// focus trapped in the menu after a click (breaking "move away to hide"), so
// this centralizes the workarounds both action toolbars need.
export function useActionsMenu() {
  const { actionsMenuOpen } = useSearchFilter();

  // Hover to open only where the device can hover. On touch, dropdown-hover
  // would suppress the menu on tap-focus, so fall back to a focus-opened one.
  const canHover = useMediaQuery("(hover: hover)");

  // Touch devices don't focus a tabindex element on tap, so the focus-based
  // dropdown never opens on its own. Force focus on tap (touch only).
  const focusOnTap = (e: MouseEvent) => {
    if (!canHover.value) (e.currentTarget as HTMLElement)?.focus();
  };

  // Clicking an item focuses the dropdown <ul>; blur it after the click so a
  // focus-opened menu (touch) closes.
  const hideMenu = (e: MouseEvent) => {
    if (e.target instanceof HTMLAnchorElement) {
      setTimeout(() => {
        if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
      }, 50);
    }
  };

  // Clicking a menu item focuses the dropdown-content <ul> (the nearest
  // focusable ancestor), which keeps the menu open via :focus-within even after
  // the pointer leaves. Release that focus on leave so the menu always closes in
  // step with actionsMenuOpen.
  const onLeave = (e: MouseEvent) => {
    actionsMenuOpen.value = false;
    const dropdown = e.currentTarget as HTMLElement;
    const active = document.activeElement as HTMLElement | null;
    if (active && dropdown.contains(active)) active.blur();
  };

  return { actionsMenuOpen, canHover, focusOnTap, hideMenu, onLeave };
}
