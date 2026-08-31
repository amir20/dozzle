/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import WelcomeModal from "./WelcomeModal.vue";

vi.mock("vue-router");

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "" },
  withBase: (path: string) => path,
}));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackLocale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      cloud: {
        welcome: {
          "turn-on": "Turn on 1 alert | Turn on {count} alerts",
          "continue-without": "Continue without alerts",
          "skip-alerts": "Not now, just send the daily findings",
          "rule-plural": "1 rule | {count} rules",
          signals: {
            exited: "Container exited with an error",
            unhealthy: "Container became unhealthy",
            oom: "Killed by the kernel (OOM)",
            restart: "Container restarted",
            cpu: "CPU over 90% for 5 minutes",
            memory: "Memory over 90% for 5 minutes",
            disk: "Volume over 85% full",
            "disk-free": "Volume under 1 GB free",
            fatal: "Any fatal line",
            error: "Any error line",
          },
        },
      },
    },
  },
});

function mountModal() {
  return mount(WelcomeModal, { global: { plugins: [i18n] } });
}

type ModalVm = { open: () => void; step: number; createdCount: number };

describe("<WelcomeModal /> starter alerts", () => {
  const pushSpy = vi.fn();

  beforeEach(() => {
    // jsdom's HTMLDialogElement lacks .close()/.showModal() — stub them so WelcomeModal's close() works.
    if (!HTMLDialogElement.prototype.close) {
      HTMLDialogElement.prototype.close = function () {};
    }
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function () {};
    }
    vi.mocked(useRouter).mockReturnValue({ push: pushSpy } as unknown as ReturnType<typeof useRouter>);
    pushSpy.mockReset();
    vi.restoreAllMocks();
  });

  function stubFetch(ruleStatus = 200) {
    const fetchMock = vi.fn(async (url: RequestInfo | URL, _init?: RequestInit) => {
      const u = String(url);
      if (u.includes("/api/notifications/dispatchers")) {
        return new Response(JSON.stringify([{ id: 7, type: "cloud", name: "Dozzle Cloud" }]), { status: 200 });
      }
      if (u.includes("/api/notifications/rules")) {
        return new Response("{}", { status: ruleStatus });
      }
      return new Response("{}", { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  async function openOnAlertsStep(wrapper: ReturnType<typeof mountModal>) {
    const vm = wrapper.vm as unknown as ModalVm;
    vm.open();
    vm.step = 2;
    await wrapper.vm.$nextTick();
  }

  function primaryCta(wrapper: ReturnType<typeof mountModal>) {
    return wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("turn on"));
  }

  test("POSTs one rule per enabled starter rule and lands on the final step", async () => {
    const fetchMock = stubFetch();
    const wrapper = mountModal();
    await openOnAlertsStep(wrapper);

    const cta = primaryCta(wrapper);
    expect(cta).toBeDefined();
    await cta!.trigger("click");
    await flushPromises();

    const ruleCalls = fetchMock.mock.calls.filter((c) => String(c[0]).includes("/api/notifications/rules"));
    // Lifecycle ships exited + unhealthy + oom (restart off), Metrics ships cpu +
    // memory + disk (absolute free space off), Logs is off as a category.
    expect(ruleCalls).toHaveLength(6);

    const bodies = ruleCalls.map((c) => JSON.parse((c[1] as RequestInit).body as string));

    const eventExpressions = bodies.map((b) => b.eventExpression).filter(Boolean);
    expect(eventExpressions).toContain('name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])');
    expect(eventExpressions).toContain('name == "health_status" && attributes["healthStatus"] == "unhealthy"');
    expect(eventExpressions).toContain('name == "oom"');
    expect(eventExpressions).not.toContain('name == "restart"');

    const metricExpressions = bodies.map((b) => b.metricExpression).filter(Boolean);
    expect(metricExpressions).toContain("cpu >= 90");
    expect(metricExpressions).toContain("memory >= 90");
    expect(metricExpressions).toContain("any(mounts, .usedPercent >= 85)");
    expect(metricExpressions).not.toContain("any(mounts, .availableBytes < 1073741824)");

    // Sustained CPU/memory must use the longest window the server allows, or the
    // "for 5 minutes" label would overstate what the rule actually checks.
    for (const b of bodies.filter(
      (x) => x.metricExpression?.startsWith("cpu") || x.metricExpression?.startsWith("memory"),
    )) {
      expect(b).toMatchObject({ cooldown: 3600, sampleWindow: 300 });
    }

    // Logs category is off by default, so no log rule is created.
    expect(bodies.map((b) => b.logExpression).filter(Boolean)).toHaveLength(0);

    const diskBody = bodies.find((b) => b.metricExpression === "any(mounts, .usedPercent >= 85)");
    expect(diskBody).toMatchObject({
      enabled: true,
      dispatcherId: 7,
      cooldown: 3600,
      sampleWindow: 15,
      containerExpression: "true",
      eventExpression: "",
      logExpression: "",
    });

    for (const b of bodies.filter((x) => x.eventExpression)) {
      expect(b).toMatchObject({
        enabled: true,
        dispatcherId: 7,
        cooldown: 0,
        sampleWindow: 0,
        containerExpression: "true",
        metricExpression: "",
        logExpression: "",
      });
    }

    // Success keeps the user in the modal so step 3 can say where alerts land.
    const vm = wrapper.vm as unknown as ModalVm;
    expect(vm.step).toBe(3);
    expect(vm.createdCount).toBe(6);
    expect(pushSpy).not.toHaveBeenCalled();
  });

  test("enabling the logs category adds its default log rule", async () => {
    const fetchMock = stubFetch();
    const wrapper = mountModal();
    await openOnAlertsStep(wrapper);

    // Third category toggle is Logs.
    const toggles = wrapper.findAll("input.toggle");
    expect(toggles).toHaveLength(3);
    await toggles[2].setValue(true);

    await primaryCta(wrapper)!.trigger("click");
    await flushPromises();

    const bodies = fetchMock.mock.calls
      .filter((c) => String(c[0]).includes("/api/notifications/rules"))
      .map((c) => JSON.parse((c[1] as RequestInit).body as string));

    // Only the fatal rule is on inside the logs category; "any error line" is off.
    expect(bodies.map((b) => b.logExpression).filter(Boolean)).toEqual(['level == "fatal"']);
    expect(bodies).toHaveLength(7);
  });

  test("skipping creates nothing and reports the final step as alert-free", async () => {
    const fetchMock = stubFetch();
    const wrapper = mountModal();
    await openOnAlertsStep(wrapper);

    const skip = wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("not now"));
    expect(skip).toBeDefined();
    await skip!.trigger("click");
    await flushPromises();

    expect(fetchMock.mock.calls.filter((c) => String(c[0]).includes("/api/notifications/rules"))).toHaveLength(0);

    const vm = wrapper.vm as unknown as ModalVm;
    expect(vm.step).toBe(3);
    expect(vm.createdCount).toBe(0);
  });

  test("falls back to ?action=create-alert when a rule POST fails", async () => {
    stubFetch(500);
    const wrapper = mountModal();
    await openOnAlertsStep(wrapper);

    await primaryCta(wrapper)!.trigger("click");
    await flushPromises();

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications", query: { action: "create-alert" } });
  });
});
