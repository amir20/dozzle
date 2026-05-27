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
          "create-alerts": "Turn on selected signals",
          signals: {
            exited: "Container exited with an error",
            "exited-desc": "Fires when a container stops with a non-zero exit code.",
            unhealthy: "Container became unhealthy",
            "unhealthy-desc": "Fires when a container's healthcheck transitions to unhealthy.",
            oom: "Container was killed by the kernel (OOM)",
            "oom-desc": "Fires when Docker reports an out-of-memory kill.",
            restart: "Container restarted",
            "restart-desc": "Off by default — noisy on its own; Cloud also uses this for loop detection.",
            disk: "Disk space running low on any volume",
            "disk-desc": "Fires when any mounted volume is over 85% full.",
          },
        },
      },
    },
  },
});

function mountModal() {
  return mount(WelcomeModal, {
    global: {
      plugins: [i18n],
    },
  });
}

describe("<WelcomeModal /> Create First Alert", () => {
  const pushSpy = vi.fn();

  beforeEach(() => {
    // jsdom's HTMLDialogElement lacks .close()/.showModal() — stub them so WelcomeModal's close() works.
    if (!HTMLDialogElement.prototype.close) {
      HTMLDialogElement.prototype.close = function () {};
    }
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function () {};
    }
    vi.mocked(useRouter).mockReturnValue({
      push: pushSpy,
    } as unknown as ReturnType<typeof useRouter>);
    pushSpy.mockReset();
    vi.restoreAllMocks();
  });

  async function openAndAdvance(wrapper: ReturnType<typeof mountModal>) {
    // open() seeds defaultOn signals
    (wrapper.vm as unknown as { open: () => void }).open();
    const vm = wrapper.vm as unknown as { step: "step1" | "step2" };
    vm.step = "step2";
    await wrapper.vm.$nextTick();
  }

  test("POSTs one rule per checked default signal and routes to /notifications", async () => {
    const fetchMock = vi.fn(async (url: RequestInfo | URL, _init?: RequestInit) => {
      const u = String(url);
      if (u.includes("/api/notifications/dispatchers")) {
        return new Response(JSON.stringify([{ id: 7, type: "cloud", name: "Dozzle Cloud" }]), { status: 200 });
      }
      if (u.includes("/api/notifications/rules")) {
        return new Response(JSON.stringify({ id: 42 }), { status: 200 });
      }
      return new Response("{}", { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const wrapper = mountModal();
    await openAndAdvance(wrapper);

    const cta = wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("turn on"));
    expect(cta).toBeDefined();
    await cta!.trigger("click");
    await flushPromises();

    const ruleCalls = fetchMock.mock.calls.filter((c) => String(c[0]).includes("/api/notifications/rules"));
    expect(ruleCalls).toHaveLength(4); // exited + unhealthy + oom + disk on by default; restart off

    const bodies = ruleCalls.map((c) => JSON.parse((c[1] as RequestInit).body as string));
    const eventExpressions = bodies.map((b) => b.eventExpression).filter(Boolean);
    expect(eventExpressions).toContain('name == "die" && !(attributes["exitCode"] in ["0", "143", "137"])');
    expect(eventExpressions).toContain('name == "health_status" && attributes["healthStatus"] == "unhealthy"');
    expect(eventExpressions).toContain('name == "oom"');
    expect(eventExpressions).not.toContain('name == "restart"');

    const metricExpressions = bodies.map((b) => b.metricExpression).filter(Boolean);
    expect(metricExpressions).toContain("any(mounts, .usedPercent >= 85)");

    // disk rule should carry its own cooldown/sampleWindow; event rules should remain at 0
    const diskBody = bodies.find((b) => b.metricExpression === "any(mounts, .usedPercent >= 85)");
    expect(diskBody).toMatchObject({
      enabled: true,
      dispatcherId: 7,
      cooldown: 3600,
      sampleWindow: 60,
      containerExpression: "true",
      eventExpression: "",
    });

    // event-based POSTs use cloud dispatcher id with no cooldown
    for (const b of bodies.filter((x) => x.eventExpression)) {
      expect(b).toMatchObject({
        enabled: true,
        dispatcherId: 7,
        cooldown: 0,
        sampleWindow: 0,
        containerExpression: "true",
        metricExpression: "",
      });
    }

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications" });
  });

  test("falls back to ?action=create-alert when POST fails", async () => {
    const fetchMock = vi.fn(async (url: RequestInfo | URL, _init?: RequestInit) => {
      const u = String(url);
      if (u.includes("/api/notifications/dispatchers")) {
        return new Response(JSON.stringify([{ id: 7, type: "cloud", name: "Dozzle Cloud" }]), { status: 200 });
      }
      if (u.includes("/api/notifications/rules")) {
        return new Response("{}", { status: 500 });
      }
      return new Response("{}", { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const wrapper = mountModal();
    await openAndAdvance(wrapper);

    const cta = wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("turn on"));
    await cta!.trigger("click");
    await flushPromises();

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications", query: { action: "create-alert" } });
  });
});
