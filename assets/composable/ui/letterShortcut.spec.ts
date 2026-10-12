/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, ref, type EffectScope } from "vue";
import { useLetterShortcut } from "./letterShortcut";

function press(key: string, init: KeyboardEventInit = {}, target: EventTarget = document.body) {
  const event = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...init });
  target.dispatchEvent(event);
  return event;
}

describe("useLetterShortcut", () => {
  let scope: EffectScope;
  let handler: ReturnType<typeof vi.fn<(e: KeyboardEvent) => void>>;
  const enabled = ref(true);

  beforeEach(() => {
    handler = vi.fn();
    enabled.value = true;
    scope = effectScope();
    scope.run(() => useLetterShortcut("s", handler, enabled));
  });

  afterEach(() => {
    scope.stop();
    document.body.innerHTML = "";
  });

  it("fires on the bare key and claims it", () => {
    const event = press("s");
    expect(handler).toHaveBeenCalledTimes(1);
    expect(event.defaultPrevented).toBe(true);
  });

  it("treats Shift as the same key", () => {
    press("S", { shiftKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("ignores other keys", () => {
    press("x");
    expect(handler).not.toHaveBeenCalled();
  });

  it.each(["metaKey", "ctrlKey", "altKey"] as const)("ignores the key with %s held", (modifier) => {
    const event = press("s", { [modifier]: true });
    expect(handler).not.toHaveBeenCalled();
    expect(event.defaultPrevented).toBe(false);
  });

  it("ignores key repeat and IME composition", () => {
    press("s", { repeat: true });
    press("s", { isComposing: true });
    expect(handler).not.toHaveBeenCalled();
  });

  it.each([
    "<input />",
    "<textarea></textarea>",
    "<select><option>a</option></select>",
    '<div contenteditable="true"></div>',
    '<div class="cm-editor"><div class="inner"></div></div>',
    '<div class="xterm"><div class="inner"></div></div>',
  ])("ignores the key while typing in %s", (html) => {
    document.body.innerHTML = html;
    const el = (document.body.querySelector(".inner") ?? document.body.firstElementChild) as HTMLElement;
    press("s", {}, el);
    expect(handler).not.toHaveBeenCalled();
  });

  it("ignores the key while a dialog is open", () => {
    document.body.innerHTML = "<dialog open></dialog>";
    press("s");
    expect(handler).not.toHaveBeenCalled();
  });

  it("follows the enabled flag on every press", () => {
    enabled.value = false;
    press("s");
    expect(handler).not.toHaveBeenCalled();
    enabled.value = true;
    press("s");
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("stops listening when its scope ends", () => {
    scope.stop();
    press("s");
    expect(handler).not.toHaveBeenCalled();
  });
});
