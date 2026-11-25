// @vitest-environment jsdom
import { describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { computed, ref } from "vue";
import { createI18n } from "vue-i18n";
import type { NotificationRule } from "@/types/notifications";
import AlertCard from "./AlertCard.vue";

vi.mock("@/composable/cloud/cloudSurface", () => ({
  useCloudSurface: () => ({ linked: ref(true) }),
}));
const { showToast } = vi.hoisted(() => ({ showToast: vi.fn() }));
vi.mock("@/composable/app/toast", () => ({ useToast: () => ({ showToast }) }));
vi.mock("@/composable/cloud/recentAlerts", () => ({
  useRecentAlerts: () => ({ forSubscription: () => computed(() => []), fetchRecentAlerts: vi.fn() }),
}));

function mountCard(rule: Partial<NotificationRule>) {
  const alert = {
    id: 1,
    name: "errors",
    enabled: true,
    containerExpression: "true",
    logExpression: 'message contains "boom"',
    dispatcher: { id: 1, name: "Dozzle Cloud", type: "cloud" },
    ...rule,
  } as NotificationRule;
  return mount(AlertCard, {
    props: { alert, dispatchers: [] },
    global: {
      plugins: [createI18n({ legacy: false, missingWarn: false, fallbackWarn: false })],
      stubs: { Popover: true },
    },
  });
}

const immediateBadge = "[data-testid=alert-immediate]";

describe("AlertCard send immediately", () => {
  it("shows on a rule set to send immediately", () => {
    expect(mountCard({ immediate: true }).find(immediateBadge).exists()).toBe(true);
  });

  it("stays hidden on a rule Dozzle Cloud reviews", () => {
    expect(mountCard({ immediate: false }).find(immediateBadge).exists()).toBe(false);
  });

  it("stays hidden once the rule has moved to a webhook", () => {
    const dispatcher = { id: 2, name: "hook", type: "webhook" } as NotificationRule["dispatcher"];
    expect(mountCard({ immediate: true, dispatcher }).find(immediateBadge).exists()).toBe(false);
  });

  it("stays hidden once the destination is deleted", () => {
    expect(mountCard({ immediate: true, dispatcher: null }).find(immediateBadge).exists()).toBe(false);
  });
});

describe("AlertCard log cooldown", () => {
  it.each(["cloud", "webhook"])("shows a cooldown for an ordinary %s log rule", (type) => {
    const dispatcher = { id: 1, name: "destination", type } as NotificationRule["dispatcher"];
    expect(mountCard({ immediate: false, cooldown: 60, dispatcher }).text()).toContain(
      "notifications.alert.cooldown-meta",
    );
  });

  it("hides the cooldown when it is off", () => {
    expect(mountCard({ cooldown: 0 }).text()).not.toContain("notifications.alert.cooldown-meta");
  });
});

describe("AlertCard toggle", () => {
  it("tells the user when the server could not save the change", async () => {
    showToast.mockClear();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ error: "could not save" }), { status: 500 })),
    );
    const card = mountCard({});
    await card.find("input.toggle").trigger("change");
    await flushPromises();
    expect(showToast).toHaveBeenCalledWith(expect.objectContaining({ type: "error", message: "could not save" }));
    vi.unstubAllGlobals();
  });
});
