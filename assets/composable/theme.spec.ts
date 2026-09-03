/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { defineComponent, h } from "vue";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { hosts: [], base: "" },
  withBase: (path: string) => path,
}));

// useColorMode's updateHTMLAttrs injects a `* { transition: none }` stylesheet into
// <head>, flips html.classList, then reads back a computed style to force a synchronous
// whole-document recalc before removing it again. Counting those three things is how we
// tell one shared instance from one per component. The style element is gone by the time
// the mount returns, so the append has to be counted as it happens.
let styleInjections = 0;
let forcedReflows = 0;
let mediaListeners = 0;

const realAppend = document.head.appendChild.bind(document.head);
document.head.appendChild = ((node: Node) => {
  if ((node as Element).tagName === "STYLE") styleInjections++;
  return realAppend(node);
}) as typeof document.head.appendChild;

const realGetComputedStyle = window.getComputedStyle.bind(window);
window.getComputedStyle = ((el: Element, ...rest: unknown[]) => {
  if (el?.tagName === "STYLE") forcedReflows++;
  return (realGetComputedStyle as any)(el, ...rest);
}) as typeof window.getComputedStyle;

window.matchMedia = ((query: string) => ({
  matches: false,
  media: query,
  onchange: null,
  addEventListener: () => mediaListeners++,
  removeEventListener: () => {},
  addListener: () => mediaListeners++,
  removeListener: () => {},
  dispatchEvent: () => false,
})) as typeof window.matchMedia;

// Imported after the probes are installed so the module-level useColorMode() is counted.
const { useResolvedTheme } = await import("./theme");
const { lightTheme } = await import("@/stores/settings");

beforeEach(() => {
  styleInjections = 0;
  forcedReflows = 0;
  mediaListeners = 0;
});

const Consumer = defineComponent({
  setup() {
    const { isDark } = useResolvedTheme();
    return () => h("span", isDark.value ? "dark" : "light");
  },
});

describe("useResolvedTheme", () => {
  test("resolves an explicit preference", () => {
    lightTheme.value = "dark";
    expect(mount(Consumer).text()).toBe("dark");

    lightTheme.value = "light";
    expect(mount(Consumer).text()).toBe("light");
  });

  test("callers share one useColorMode instance", () => {
    // Regression guard for #4991. Calling useColorMode() inside the composable gave every
    // ContainerIcon -- one per container row, menu entry and search hit -- its own
    // matchMedia listener and two forced full-document reflows at mount, so a host with
    // 77 containers paid 154 of them on every render pass.
    const wrapper = mount(
      defineComponent({
        render: () =>
          h(
            "div",
            Array.from({ length: 50 }, (_, i) => h(Consumer, { key: i })),
          ),
      }),
    );

    expect(styleInjections).toBe(0);
    expect(forcedReflows).toBe(0);
    expect(mediaListeners).toBe(0);

    wrapper.unmount();
  });
});
