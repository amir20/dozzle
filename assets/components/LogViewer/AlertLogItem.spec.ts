/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import AlertLogItem from "./AlertLogItem.vue";
import { AlertLogEntry } from "@/models/LogEntry";
import type { CloudAlert } from "@/composable/cloudAlerts";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

const ns = (n: number) => n * 1_000_000;

function mountAlert(overrides: Partial<CloudAlert> = {}) {
  const alert: CloudAlert = {
    alertId: 1,
    containerId: "abc",
    hostId: "h",
    ts: ns(1_000_000),
    headline: "Pool exhausted",
    level: "error",
    eventCount: 47,
    createdAt: ns(1_000_000),
    isOrigin: true,
    ...overrides,
  };
  return mount(AlertLogItem, {
    props: { logEntry: new AlertLogEntry(alert, new Date(alert.ts / 1_000_000)) },
    global: {
      stubs: { LogItem: { template: "<div><slot /></div>" } },
      mocks: { $t: (key: string, arg?: unknown) => `${key}:${JSON.stringify(arg)}` },
    },
  });
}

describe("<AlertLogItem />", () => {
  test("renders a full card at the origin", () => {
    const wrapper = mountAlert();
    expect(wrapper.text()).toContain("Pool exhausted");
    expect(wrapper.text()).toContain("label.alert-events");
    expect(wrapper.find("[data-origin='true']").exists()).toBe(true);
  });

  // A long incident would bury the logs if every window it touched drew a full
  // card, so follow-ups get one quiet line instead.
  test("renders a compact line at a follow-up anchor", () => {
    const wrapper = mountAlert({ isOrigin: false });
    expect(wrapper.text()).toContain("label.alert-still-firing");
    expect(wrapper.text()).not.toContain("label.alert-events");
  });

  // alerts.level is free text and defaults to '' for anything that never went
  // through a summarizer. A neutral fallback made the common case look like an
  // ordinary log line, which is the one thing this row must not look like.
  describe("level styling", () => {
    test("treats an empty level as an error", () => {
      expect(mountAlert({ level: "" }).find("[data-level='error']").exists()).toBe(true);
    });

    test("treats an unrecognised level as an error", () => {
      expect(mountAlert({ level: "mystery" }).find("[data-level='error']").exists()).toBe(true);
    });

    test("keeps warn and info distinct", () => {
      expect(mountAlert({ level: "warning" }).find("[data-level='warn']").exists()).toBe(true);
      expect(mountAlert({ level: "debug" }).find("[data-level='info']").exists()).toBe(true);
    });
  });

  test("shows the container count only when the incident is wider than one", () => {
    expect(mountAlert({ containerCount: 1 }).text()).not.toContain("label.alert-containers");
    expect(mountAlert({ containerCount: 3 }).text()).toContain("label.alert-containers");
  });

  test("shows held-back count only when triage suppressed follow-ups", () => {
    expect(mountAlert({ suppressedCount: 0 }).text()).not.toContain("label.alert-held-back");
    expect(mountAlert({ suppressedCount: 34 }).text()).toContain("label.alert-held-back");
  });

  test("links out to cloud only when the key has an app url", () => {
    expect(mountAlert({ url: undefined }).find("a").exists()).toBe(false);
    expect(mountAlert({ url: "https://app.example.com/alerts/1" }).find("a").attributes("href")).toBe(
      "https://app.example.com/alerts/1",
    );
  });
});
