/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
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
  default: { cloudUrl: "https://cloud.example", hosts: [] },
  withBase: (p: string) => p,
}));
vi.mock("@/composable/ui/media", async () => {
  const { ref } = await import("vue");
  return { isMobile: ref(false), canHover: ref(true), isMac: true };
});
vi.mock("@/composable/cloud/cloudRail", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/composable/cloud/cloudRail")>();
  return {
    cloudPanelHref: actual.cloudPanelHref,
    useCloudRail: () => ({ panel, railOffset: ref(48), openRail, closeRail }),
    useRailItems: () =>
      computed(() =>
        (["findings", "alerts", "chat", "metrics"] as const).map((id) => ({ id, label: id, icon: "i", dot: false })),
      ),
  };
});
vi.mock("@/composable/cloud/cloudScope", () => ({
  narrowBellWiden: vi.fn(),
  useCloudScope: () => ({ inView, scoped, containers: computed(() => (scoped.value ? inView.value : [])) }),
}));
vi.mock("@/composable/cloud/cloudChat", () => ({ useCloudChat: () => ({ messages: ref([]), reset: vi.fn() }) }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: { cloud: { title: "Dozzle Cloud" }, "cloud-rail": { "open-account": "Plan and billing" } } },
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

const mountDrawer = () => mount(CloudDrawer, { attachTo: document.body, global: { plugins: [i18n], stubs } });

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

  test("the footer links to the container in Cloud, and to the panel's list once widened", async () => {
    const wrapper = mountDrawer();
    expect(wrapper.find("footer a").attributes("href")).toBe(
      "https://cloud.example/metrics/h1/api?from=dozzle_drawer_findings",
    );

    scoped.value = false;
    panel.value = "alerts";
    await nextTick();
    expect(wrapper.find("footer a").attributes("href")).toBe("https://cloud.example/alerts?from=dozzle_drawer_alerts");
    // Widened, metrics are every host's.
    panel.value = "metrics";
    await nextTick();
    expect(wrapper.findComponent({ name: "RailHostMetrics" }).exists()).toBe(true);
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
  // drawer, no scope to pick, and the footer goes to plan and billing.
  test("the account panel has its own title, no scope chip, and links to billing", () => {
    panel.value = "account";
    const wrapper = mountDrawer();
    expect(wrapper.findComponent({ name: "RailAccount" }).exists()).toBe(true);
    expect(wrapper.find("h2").text()).toBe("Dozzle Cloud");
    expect(wrapper.findComponent({ name: "CloudScopeChip" }).exists()).toBe(false);
    expect(wrapper.find("footer a").attributes("href")).toBe(
      "https://cloud.example/settings?from=dozzle_drawer_account",
    );
    expect(wrapper.find("footer").text()).toBe("Plan and billing");
  });
});
