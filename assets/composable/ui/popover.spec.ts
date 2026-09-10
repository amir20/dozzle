/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { effectScope, ref } from "vue";
import { useAnchoredPopover } from "./popover";

/**
 * jsdom ships no popover API, so the panel is a stand-in that models the parts
 * this composable leans on: showPopover throwing when already open, and
 * `beforetoggle` firing synchronously from inside the call.
 *
 * `toggle` is deliberately NOT fired. The browser delivers it on a rendering
 * update, which is exactly the delivery a document that is not rendering never
 * gets, and the reveal has to survive that.
 */
function fakePanel(onBeforeToggle: (e: ToggleEvent) => void) {
  const el = document.createElement("div");
  let open = false;
  Object.assign(el, {
    showPopover() {
      if (open) throw new DOMException("already open", "InvalidStateError");
      onBeforeToggle({ oldState: "closed", newState: "open" } as ToggleEvent);
      open = true;
    },
    hidePopover() {
      open = false;
    },
  });
  const matches = el.matches.bind(el);
  el.matches = ((selector: string) =>
    selector === ":popover-open" ? open : matches(selector)) as HTMLElement["matches"];
  document.body.append(el);
  return el;
}

function withPopover(fn: (api: ReturnType<typeof useAnchoredPopover>, panel: HTMLElement) => void) {
  const scope = effectScope();
  const anchor = ref<HTMLElement | null>(document.createElement("div"));
  const panel = ref<HTMLElement | null>(null);
  const api = scope.run(() => useAnchoredPopover(anchor, panel))!;
  panel.value = fakePanel(api.onBeforeToggle);
  try {
    fn(api, panel.value);
  } finally {
    scope.stop();
  }
}

describe("useAnchoredPopover", () => {
  test("reveals the panel without waiting for a toggle event", () => {
    withPopover(({ show }, panel) => {
      show();
      expect(panel.matches(":popover-open")).toBe(true);
      // Hidden by beforeToggle for the unmeasurable first pass, cleared again
      // by the time show() returns.
      expect(panel.style.visibility).toBe("");
    });
  });

  test("a missed toggle event does not wedge it shut", () => {
    withPopover(({ show, hide }, panel) => {
      show();
      hide();
      expect(panel.matches(":popover-open")).toBe(false);
      show();
      expect(panel.matches(":popover-open")).toBe(true);
      expect(panel.style.visibility).toBe("");
    });
  });

  test("show on an already open panel is a no-op rather than a throw", () => {
    withPopover(({ show }, panel) => {
      show();
      expect(() => show()).not.toThrow();
      expect(panel.matches(":popover-open")).toBe(true);
    });
  });
});
