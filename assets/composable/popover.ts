import type { Ref } from "vue";

export type PopoverPlacement = "bottom-start" | "bottom-end" | "right-start" | "right-end";

/** Viewport breathing room kept around a panel. */
const MARGIN = 8;

/**
 * Drives a native popover panel anchored to a trigger element.
 *
 * The panel lives in the top layer, so nothing in the page can clip it: an ancestor with
 * `overflow` (the container table's rounded card, the log list's scroller) used to cut menus
 * in half. The top layer has no anchor of its own, so the panel is placed against the
 * viewport by hand. CSS anchor positioning would do this, but Firefox does not have it yet.
 */
export function useAnchoredPopover(
  anchor: Ref<HTMLElement | null>,
  panel: Ref<HTMLElement | null>,
  options: {
    placement?: () => PopoverPlacement;
    /** Space between the trigger and the panel. Hover menus want this small to cross. */
    gap?: () => number;
  } = {},
) {
  const isOpen = ref(false);
  // A click on the trigger of an open popover is an outside click, so the browser light
  // dismisses it before the click handler runs. Without this the menu would reopen at once
  // and never close from its own button.
  let closedAt = 0;

  const placement = () => options.placement?.() ?? "bottom-start";
  const gap = () => options.gap?.() ?? 4;

  function position() {
    if (!anchor.value || !panel.value) return;
    const rect = anchor.value.getBoundingClientRect();
    const { offsetWidth: width, offsetHeight: height } = panel.value;

    const [side, align] = placement().split("-");
    let left: number;
    let top: number;

    if (side === "right") {
      left = rect.right + gap();
      if (left + width > window.innerWidth - MARGIN) left = rect.left - width - gap();
      top = align === "end" ? rect.bottom - height : rect.top;
    } else {
      left = align === "end" ? rect.right - width : rect.left;
      top = rect.bottom + gap();
      // Flip above only when there is actually more room up there.
      if (top + height > window.innerHeight - MARGIN && rect.top > height) top = rect.top - height - gap();
    }

    const clamp = (value: number, max: number) => Math.min(Math.max(MARGIN, value), Math.max(MARGIN, max - MARGIN));
    panel.value.style.left = `${clamp(left, window.innerWidth - width)}px`;
    panel.value.style.top = `${clamp(top, window.innerHeight - height)}px`;
  }

  function onBeforeToggle(event: ToggleEvent) {
    if (event.newState !== "open" || !panel.value) return;
    // The panel is still display:none here, so it cannot be measured and this first pass can
    // only be a guess. Keeping it invisible until `toggle` avoids showing that guess.
    panel.value.style.visibility = "hidden";
    position();
  }

  function onToggle(event: ToggleEvent) {
    isOpen.value = event.newState === "open";
    if (isOpen.value) {
      // Now that it is laid out its real size is known, so it can flip and clamp.
      position();
      if (panel.value) panel.value.style.visibility = "";
      window.addEventListener("scroll", position, true);
      window.addEventListener("resize", position);
    } else {
      closedAt = performance.now();
      stopTracking();
    }
  }

  function stopTracking() {
    window.removeEventListener("scroll", position, true);
    window.removeEventListener("resize", position);
  }

  const show = () => {
    if (!isOpen.value) panel.value?.showPopover();
  };
  const hide = () => {
    if (isOpen.value) panel.value?.hidePopover();
  };
  const toggle = () => {
    if (isOpen.value) hide();
    else if (performance.now() - closedAt > 250) show();
  };

  onScopeDispose(stopTracking);

  return { isOpen, position, onBeforeToggle, onToggle, show, hide, toggle };
}
