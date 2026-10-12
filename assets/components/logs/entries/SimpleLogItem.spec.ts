/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import SimpleLogItem from "./SimpleLogItem.vue";
import GroupedLogItem from "./GroupedLogItem.vue";
import { GroupedLogEntry, SimpleLogEntry } from "@/models/LogEntry";
import { showTimestamp } from "@/stores/settings";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

const stubs = { LogItem: { template: "<div><slot /></div>" }, LogLevel: true };
const line = "\x1b[90m2026-09-13T22:28:56Z\x1b[0m INF ready";
const tp = line.indexOf("INF");

afterEach(() => {
  showTimestamp.value = true;
});

describe("<SimpleLogItem /> timestamp prefix", () => {
  const entry = new SimpleLogEntry(line, "c1", 1, new Date(), "info", "stdout", line, tp);

  test("hides the app's timestamp while the date column shows it", () => {
    showTimestamp.value = true;
    const wrapper = mount(SimpleLogItem, { props: { logEntry: entry }, global: { stubs } });
    expect(wrapper.find(".log-message").text()).toBe("INF ready");
  });

  test("keeps the app's timestamp when the date column is off", () => {
    showTimestamp.value = false;
    const wrapper = mount(SimpleLogItem, { props: { logEntry: entry }, global: { stubs } });
    expect(wrapper.find(".log-message").text()).toBe("2026-09-13T22:28:56Z INF ready");
  });
});

describe("<GroupedLogItem /> timestamp prefix", () => {
  test("slices each line by its own prefix", () => {
    showTimestamp.value = true;
    const entry = new GroupedLogEntry([line, "  at main.go:1"], "c1", 1, new Date(), "error", "stderr", [tp, 0]);
    const wrapper = mount(GroupedLogItem, { props: { logEntry: entry }, global: { stubs } });
    expect(wrapper.findAll(".log-message").map((m) => m.text())).toEqual(["INF ready", "at main.go:1"]);
  });
});

// What the backend sends: HTML already escaped, with <mark> and <a> spliced in, plus raw ANSI.
const rich =
  '\x1b[31mred\x1b[0m &lt;b&gt;tag&lt;/b&gt; a &amp; b <mark>hit</mark> <a href="https://x.com/?a=1&amp;b=2" target="_blank">https://x.com</a>';

// The red run may carry its color inline or as a class backed by main.css; either names the variable.
const colorOf = (el: Element, text: string) => {
  const span = [...el.querySelectorAll("span")].find((s) => s.textContent === text);
  return span ? `${span.getAttribute("style") ?? ""} ${span.className}` : "";
};

const expectRich = (el: Element) => {
  expect(el.textContent).toBe("red <b>tag</b> a & b hit https://x.com");
  expect(el.querySelector("b")).toBeNull();
  expect(el.querySelector("mark")?.textContent).toBe("hit");
  expect(el.querySelector("a")?.getAttribute("href")).toBe("https://x.com/?a=1&b=2");
  expect(colorOf(el, "red")).toMatch(/ansi-red/);
};

describe("<SimpleLogItem /> ANSI rendering", () => {
  test("colors ANSI without escaping the backend's markup again", () => {
    const entry = new SimpleLogEntry(rich, "c1", 1, new Date(), "info", "stdout", rich);
    const wrapper = mount(SimpleLogItem, { props: { logEntry: entry }, global: { stubs } });
    expectRich(wrapper.find(".log-message").element);
  });
});

describe("<GroupedLogItem /> ANSI rendering", () => {
  test("colors each line and keeps its markup, with no color leaking between lines", () => {
    const entry = new GroupedLogEntry([rich, "\x1b[32mopen", "after"], "c1", 1, new Date(), "error", "stderr");
    const wrapper = mount(GroupedLogItem, { props: { logEntry: entry }, global: { stubs } });
    const lines = wrapper.findAll(".log-message");
    expectRich(lines[0].element);
    expect(colorOf(lines[1].element, "open")).toMatch(/ansi-green/);
    expect(lines[2].element.innerHTML).toBe("after");
  });
});
