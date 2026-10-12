import { useEventListener } from "@vueuse/core";
import { toValue, type MaybeRefOrGetter } from "vue";

// Anywhere a letter is text, not a command: form fields, CodeMirror, a terminal.
const TYPING_SURFACE = 'input, textarea, select, [contenteditable]:not([contenteditable="false"]), .cm-editor, .xterm';

function isTypingTarget(el: EventTarget | Element | null): boolean {
  return el instanceof Element && (el.closest(TYPING_SURFACE) !== null || (el as HTMLElement).isContentEditable);
}

/**
 * Whether a bare-key shortcut may act on this keydown. Any chord, a repeat, an IME
 * composition, focus in something that takes text, or an open dialog (drawers, the
 * palette and every modal are a `<dialog>`) means the key belongs to someone else.
 */
export function isBareKeyFree(e: KeyboardEvent): boolean {
  if (e.defaultPrevented || e.metaKey || e.ctrlKey || e.altKey) return false;
  if (e.isComposing || e.repeat) return false;
  if (isTypingTarget(e.target) || isTypingTarget(document.activeElement)) return false;
  if (document.querySelector("dialog[open]")) return false;
  return true;
}

/**
 * Runs `handler` when `key` is pressed on its own (Shift allowed, so `S` counts as `s`).
 * `enabled` is read on every press, so a shortcut can follow state without re-registering.
 */
export function useLetterShortcut(
  key: string,
  handler: (e: KeyboardEvent) => void,
  enabled: MaybeRefOrGetter<boolean> = true,
) {
  const wanted = key.toLowerCase();
  return useEventListener(window, "keydown", (e: KeyboardEvent) => {
    if (e.key?.toLowerCase() !== wanted) return;
    if (!toValue(enabled) || !isBareKeyFree(e)) return;
    e.preventDefault();
    handler(e);
  });
}
