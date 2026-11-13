/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { defineComponent } from "vue";
import { createI18n } from "vue-i18n";
import AlertDot from "./AlertDot.vue";
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { base: "", hosts: [] }, withBase: (p: string) => p }));
vi.mock("@/stores/container", () => ({ useContainerStore: () => ({ allContainersById: { here: {} } }) }));
vi.mock("@/composable/logs/logJump", () => ({ useLogJump: () => ({ jumpTo: vi.fn() }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ byContainer: { value: new Map() }, fetchRecentAlerts: vi.fn() }),
}));

const alert = (containerId: string) =>
  ({ alertId: "a", containerId, hostId: "h", ts: 1e15, headline: "OOM", level: "error", eventCount: 1 }) as CloudAlert;

function render(containerId: string) {
  return mount(AlertDot, {
    props: { alert: alert(containerId) },
    global: {
      plugins: [createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false })],
      stubs: {
        Popover: defineComponent({ template: "<div><slot name='trigger' /><slot :close='() => {}' /></div>" }),
        RelativeTime: true,
      },
    },
  });
}

describe("<AlertDot />", () => {
  test("offers the lines while the alert's container is here", () => {
    expect(render("here").findAll("button").length).toBe(2);
  });

  // A deploy removes the container it replaced, and the metrics timeline shows
  // alerts from those too. Jumping to one lands on Not Found.
  test("offers no jump once the alert's container is gone", () => {
    expect(render("gone").findAll("button").length).toBe(1);
  });
});
