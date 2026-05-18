/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
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
          "default-alert-name": "Container exited with error",
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
    // allow async fetch chain to settle
    for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0));
    await wrapper.vm.$nextTick();

    const ruleCalls = fetchMock.mock.calls.filter((c) => String(c[0]).includes("/api/notifications/rules"));
    expect(ruleCalls).toHaveLength(3); // exited + unhealthy + oom on by default; restart off

    const expressions = ruleCalls.map((c) => JSON.parse((c[1] as RequestInit).body as string).eventExpression);
    expect(expressions).toContain('name == "die" && attributes["exitCode"] != "0"');
    expect(expressions).toContain('name == "health_status" && attributes["healthStatus"] == "unhealthy"');
    expect(expressions).toContain('name == "oom"');
    expect(expressions).not.toContain('name == "restart"');

    // every POST uses cloud dispatcher id
    for (const c of ruleCalls) {
      const body = JSON.parse((c[1] as RequestInit).body as string);
      expect(body).toMatchObject({
        enabled: true,
        dispatcherId: 7,
        cooldown: 0,
        sampleWindow: 0,
        containerExpression: "true",
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
    for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0));
    await wrapper.vm.$nextTick();

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications", query: { action: "create-alert" } });
  });
});
