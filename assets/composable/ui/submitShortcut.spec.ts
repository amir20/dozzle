/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, ref, type EffectScope } from "vue";
import { useSubmitShortcut } from "./submitShortcut";

function press(init: KeyboardEventInit = {}, target: EventTarget = document.activeElement ?? document.body) {
  const event = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true, ...init });
  target.dispatchEvent(event);
  return event;
}

function openDialog(): HTMLDialogElement {
  const dialog = document.createElement("dialog");
  dialog.setAttribute("open", "");
  document.body.appendChild(dialog);
  return dialog;
}

describe("useSubmitShortcut", () => {
  let scope: EffectScope;
  let handler: ReturnType<typeof vi.fn<(e: KeyboardEvent) => void>>;
  let drawer: HTMLDialogElement;
  let root: HTMLElement;
  let input: HTMLInputElement;
  const enabled = ref(true);

  beforeEach(() => {
    handler = vi.fn();
    enabled.value = true;
    drawer = openDialog();
    root = document.createElement("div");
    input = document.createElement("input");
    root.appendChild(input);
    drawer.appendChild(root);
    input.focus();
    scope = effectScope();
    scope.run(() => useSubmitShortcut(root, handler, enabled));
  });

  afterEach(() => {
    scope.stop();
    document.body.innerHTML = "";
  });

  it("fires on ⌘↵ and Ctrl+↵, even from a text field", () => {
    expect(press({ metaKey: true }).defaultPrevented).toBe(true);
    press({ ctrlKey: true });
    expect(handler).toHaveBeenCalledTimes(2);
  });

  it("claims the chord before an editor inside the drawer sees it", () => {
    const editor = vi.fn();
    input.addEventListener("keydown", editor);
    press({ metaKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
    expect(editor).not.toHaveBeenCalled();
  });

  it("lets the chord through untouched while disabled", () => {
    const editor = vi.fn();
    input.addEventListener("keydown", editor);
    enabled.value = false;
    press({ metaKey: true });
    expect(editor).toHaveBeenCalledTimes(1);
  });

  it("ignores a bare Enter and other chords", () => {
    press();
    press({ metaKey: true, shiftKey: true });
    press({ metaKey: true, altKey: true });
    press({ metaKey: true, repeat: true });
    expect(handler).not.toHaveBeenCalled();
  });

  it("follows enabled", () => {
    enabled.value = false;
    press({ metaKey: true });
    expect(handler).not.toHaveBeenCalled();
  });

  it("still fires when focus fell to the body", () => {
    input.blur();
    press({ metaKey: true }, document.body);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("leaves the chord to a dialog opened on top", () => {
    const palette = openDialog();
    const field = document.createElement("input");
    palette.appendChild(field);
    field.focus();
    press({ metaKey: true });
    field.blur();
    press({ metaKey: true }, document.body);
    expect(handler).not.toHaveBeenCalled();
  });
});
