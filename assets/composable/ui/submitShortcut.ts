import { useEventListener } from "@vueuse/core";
import { toValue, type MaybeRefOrGetter } from "vue";

// Clicking plain text in a dialog drops focus to <body>, so the body counts as the
// dialog's own focus as long as nothing else is open beside it.
function ownsFocus(dialog: HTMLDialogElement | null): boolean {
  const active = document.activeElement;
  const open = document.querySelectorAll("dialog[open]");
  if (!dialog) return open.length === 0;
  if (dialog.contains(active)) return true;
  return (active === null || active === document.body) && open.length === 1 && open[0] === dialog;
}

/**
 * Runs `handler` on ⌘↵ (Ctrl+↵ off a Mac): the "do it" key for the primary action of
 * a drawer or dialog. Unlike a bare-letter shortcut it fires from inside a text field,
 * since the chord is never text. It listens in the capture phase and stops the event,
 * because CodeMirror's default keymap binds ⌘↵ to "insert blank line" and would
 * otherwise put a newline in a single-line expression before the form ever heard it.
 *
 * `scope` is an element inside the surface that owns the action. The press only counts
 * while focus sits in the same `<dialog>` as that element, so a palette or modal opened
 * on top keeps ⌘↵ for itself. `enabled` is read on every press.
 */
export function useSubmitShortcut(
  scope: MaybeRefOrGetter<HTMLElement | null | undefined>,
  handler: (e: KeyboardEvent) => void,
  enabled: MaybeRefOrGetter<boolean> = true,
) {
  return useEventListener(
    window,
    "keydown",
    (e: KeyboardEvent) => {
      if (e.key !== "Enter" || !(e.metaKey || e.ctrlKey) || e.altKey || e.shiftKey) return;
      if (e.defaultPrevented || e.repeat || e.isComposing) return;
      const el = toValue(scope);
      if (!el?.isConnected || !toValue(enabled)) return;
      if (!ownsFocus(el.closest("dialog"))) return;
      e.preventDefault();
      e.stopPropagation();
      handler(e);
    },
    { capture: true },
  );
}
