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
  messages: { en: { cloud: { welcome: { "default-alert-name": "Container exited with error" } } } },
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

  test("POSTs default rule with cloud dispatcher id and routes to /notifications?highlight=<id>", async () => {
    const fetchMock = vi.fn(async (url: RequestInfo | URL) => {
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
    const vm = wrapper.vm as unknown as { step: "step1" | "step2" } & Record<string, unknown>;
    // jump to step2 to expose the CTA
    vm.step = "step2";
    await wrapper.vm.$nextTick();

    const cta = wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("create"));
    expect(cta).toBeDefined();
    await cta!.trigger("click");
    // allow async fetch chain to settle
    await new Promise((r) => setTimeout(r, 0));
    await new Promise((r) => setTimeout(r, 0));
    await wrapper.vm.$nextTick();

    const postCall = fetchMock.mock.calls.find((c) => String(c[0]).includes("/api/notifications/rules"));
    expect(postCall).toBeDefined();
    const body = JSON.parse((postCall![1] as RequestInit).body as string);
    expect(body).toMatchObject({
      enabled: true,
      dispatcherId: 7,
      eventExpression: 'name == "die" && attributes["exitCode"] != "0"',
      cooldown: 0,
      sampleWindow: 0,
    });

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications", query: { highlight: "42" } });
  });

  test("falls back to ?action=create-alert when POST fails", async () => {
    const fetchMock = vi.fn(async (url: RequestInfo | URL) => {
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
    const vm = wrapper.vm as unknown as { step: "step1" | "step2" } & Record<string, unknown>;
    vm.step = "step2";
    await wrapper.vm.$nextTick();

    const cta = wrapper.findAll("button").find((b) => b.text().toLowerCase().includes("create"));
    await cta!.trigger("click");
    await new Promise((r) => setTimeout(r, 0));
    await new Promise((r) => setTimeout(r, 0));
    await wrapper.vm.$nextTick();

    expect(pushSpy).toHaveBeenCalledWith({ path: "/notifications", query: { action: "create-alert" } });
  });
});
