/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import { defineComponent } from "vue";
import { createI18n } from "vue-i18n";
import MetricsTimeline from "./MetricsTimeline.vue";
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";
import type { RecentDeploy } from "@/composable/cloud/recentDeploys";

const HOUR = 3_600_000;

const alert = (ms: number, id = "a1"): CloudAlert =>
  ({
    alertId: id,
    containerId: "c1",
    hostId: "h",
    ts: ms * 1e6,
    headline: "OOM",
    level: "error",
    eventCount: 1,
  }) as CloudAlert;
const deploy = (ms: number): RecentDeploy =>
  ({
    host: "h",
    name: "api",
    oldId: "o",
    newId: "n",
    imageRef: "api:2",
    at: new Date(ms).toISOString(),
  }) as RecentDeploy;

function render(props: { alerts?: CloudAlert[]; deploys?: RecentDeploy[] }) {
  return mount(MetricsTimeline, {
    props: { from: 0, to: 10 * HOUR, alerts: [], deploys: [], format: String, ...props },
    global: {
      plugins: [createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false })],
      stubs: {
        AlertDot: defineComponent({ props: ["alert"], template: "<i data-alert-dot />" }),
        Popover: defineComponent({ template: "<div><slot name='trigger' /><slot /></div>" }),
      },
    },
  });
}

const lefts = (wrapper: ReturnType<typeof render>, kind: string) =>
  wrapper.findAll(`[data-mark="${kind}"]`).map((m) => m.attributes("style"));

describe("<MetricsTimeline />", () => {
  test("draws nothing when nothing happened in the window", () => {
    expect(render({}).find("[data-timeline]").exists()).toBe(false);
  });

  // The strip shares the bars' time axis, so a mark sits over the bar it fell in.
  test("places each mark at its moment on the window's axis", () => {
    const wrapper = render({ alerts: [alert(2.5 * HOUR)], deploys: [deploy(5 * HOUR)] });

    expect(lefts(wrapper, "alert")).toEqual(["left: 25%;"]);
    expect(lefts(wrapper, "deploy")).toEqual(["left: 50%;"]);
  });

  test("leaves out what fell outside the window", () => {
    const wrapper = render({ alerts: [alert(-HOUR), alert(11 * HOUR, "a2")], deploys: [deploy(12 * HOUR)] });
    expect(wrapper.find("[data-timeline]").exists()).toBe(false);
  });

  // A deploy is a ring and an alert a solid dot, so the two never read alike.
  test("an alert is an AlertDot and a deploy is a ring", () => {
    const wrapper = render({ alerts: [alert(HOUR)], deploys: [deploy(2 * HOUR)] });

    expect(wrapper.find('[data-mark="alert"] [data-alert-dot]').exists()).toBe(true);
    expect(wrapper.find('[data-mark="deploy"] .deploy-mark').exists()).toBe(true);
  });
});
