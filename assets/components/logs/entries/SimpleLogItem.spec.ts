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
