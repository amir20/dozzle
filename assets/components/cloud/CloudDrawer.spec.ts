/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount, RouterLinkStub } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, nextTick, ref } from "vue";
import { createI18n } from "vue-i18n";
import CloudDrawer from "./CloudDrawer.vue";
import { isMobile } from "@/composable/ui/media";

// Writable in this mock, read-only in the real media query.
const mobile = isMobile as unknown as { value: boolean };

type Panel = "findings" | "alerts" | "chat" | "metrics" | "account";
const panel = ref<Panel | undefined>("findings");
const scoped = ref(true);
const inView = ref([{ id: "a1", name: "api", host: "h1" }]);
const closeRail = vi.fn(() => (panel.value = undefined));
const openRail = vi.fn((id: Panel) => (panel.value = id));

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { cloudUrl: "https://cloud.example", hosts: [], enableNotifications: true },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/ui/media", async () => {
  const { ref } = await import("vue");
  return { isMobile: ref(false), canHover: ref(true), isMac: true };
});
vi.mock("@/composable/cloud/cloudRail", () => {
  return {
    useCloudRail: () => ({ panel, railOffset: ref(48), openRail, closeRail }),
    useRailItems: () =>
      computed(() =>
        (["findings", "alerts", "chat", "metrics"] as const).map((id) => ({ id, label: id, icon: "i", dot: false })),
      ),
  };
});
vi.mock("@/composable/cloud/cloudScope", () => ({
  useCloudScope: () => ({ inView, scoped, containers: computed(() => (scoped.value ? inView.value : [])) }),
}));
vi.mock("@/composable/cloud/cloudChat", () => ({ useCloudChat: () => ({ messages: ref([]), reset: vi.fn() }) }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: { cloud: { title: "Dozzle Cloud" } } },
});

const stubs = {
  ChatPane: true,
  RailFindings: true,
  RailAlerts: true,
  RailMetrics: true,
  RailHostMetrics: true,
  CloudScopeChip: true,
  CloudMark: true,
  RailAccount: true,
  Transition: false,
};

const mountDrawer = () =>
  mount(CloudDrawer, {
    attachTo: document.body,
    global: { plugins: [i18n], stubs: { ...stubs, RouterLink: RouterLinkStub } },
  });

afterEach(() => {
  closeRail.mockClear();
  panel.value = "findings";
  scoped.value = true;
  mobile.value = false;
  document.body.innerHTML = "";
});

describe("<CloudDrawer />", () => {
  test("has no tab row of its own on a desktop, where the rail is the tabs", () => {
    const wrapper = mountDrawer();
    expect(wrapper.findComponent({ name: "RailFindings" }).exists()).toBe(true);
    expect(wrapper.find("nav").exists()).toBe(false);
    expect(wrapper.findComponent({ name: "CloudScopeChip" }).exists()).toBe(true);
  });

  // Widened, metrics are every host's.
  test("metrics widened to the instance show every host", async () => {
    const wrapper = mountDrawer();
    scoped.value = false;
    panel.value = "metrics";
    await nextTick();
    expect(wrapper.findComponent({ name: "RailHostMetrics" }).exists()).toBe(true);
    // Nothing links out at the bottom: each panel links to Cloud where it has something to open.
    expect(wrapper.find("footer").exists()).toBe(false);
  });

  // What fired is in the drawer; what makes it fire is one button away.
  test("the Alerts panel has a button to the rules, and only that panel", async () => {
    panel.value = "alerts";
    const wrapper = mountDrawer();
    const rules = wrapper.findComponent<typeof RouterLinkStub>('[data-testid="alert-rules"]');
    expect(rules.props("to")).toEqual({ name: "/notifications" });
    await rules.trigger("click");
    expect(closeRail).toHaveBeenCalled();

    panel.value = "findings";
    await nextTick();
    expect(wrapper.find('[data-testid="alert-rules"]').exists()).toBe(false);
  });

  test("the backdrop and Esc close it", async () => {
    const wrapper = mountDrawer();
    await wrapper.find(".fixed.inset-0").trigger("click");
    expect(closeRail).toHaveBeenCalledTimes(1);

    panel.value = "chat";
    await nextTick();
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    expect(closeRail).toHaveBeenCalledTimes(2);
  });

  test("Esc leaves a modal on top of it alone", async () => {
    mountDrawer();
    const open = document.createElement("dialog");
    open.setAttribute("open", "");
    document.body.appendChild(open);
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    expect(closeRail).not.toHaveBeenCalled();
  });

  test("is a labelled modal dialog", () => {
    const section = mountDrawer().find("section");
    expect(section.attributes("role")).toBe("dialog");
    expect(section.attributes("aria-modal")).toBe("true");
    expect(section.attributes("aria-label")).toBe("findings");
  });

  test("takes focus on open and hands it back to what opened it on close", async () => {
    panel.value = undefined;
    const opener = document.createElement("button");
    document.body.appendChild(opener);
    opener.focus();
    const wrapper = mountDrawer();

    panel.value = "findings";
    await flushPromises();
    expect(document.activeElement).toBe(wrapper.find("section").element);

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    await flushPromises();
    expect(document.activeElement).toBe(opener);
  });

  test("leaves focus where the reader put it after closing", async () => {
    panel.value = undefined;
    const opener = document.createElement("button");
    const elsewhere = document.createElement("input");
    document.body.append(opener, elsewhere);
    opener.focus();
    mountDrawer();
    panel.value = "alerts";
    await flushPromises();

    elsewhere.focus();
    panel.value = undefined;
    await flushPromises();
    expect(document.activeElement).toBe(elsewhere);
  });

  test("a phone gets the four tabs as a bar inside the sheet, and the account mark", async () => {
    mobile.value = true;
    const wrapper = mountDrawer();
    const tabs = wrapper.findAll("nav button");
    expect(tabs.map((b) => b.text())).toEqual(["findings", "alerts", "chat", "metrics"]);
    expect(wrapper.findComponent({ name: "CloudMark" }).exists()).toBe(true);

    await tabs[2].trigger("click");
    expect(openRail).toHaveBeenCalledWith("chat");
  });

  // The cloud mark opens the account here rather than in a popover: same
  // drawer, with no scope to pick.
  test("the account panel has its own title and no scope chip", () => {
    panel.value = "account";
    const wrapper = mountDrawer();
    expect(wrapper.findComponent({ name: "RailAccount" }).exists()).toBe(true);
    expect(wrapper.find("h2").text()).toBe("Dozzle Cloud");
    expect(wrapper.findComponent({ name: "CloudScopeChip" }).exists()).toBe(false);
  });
});
