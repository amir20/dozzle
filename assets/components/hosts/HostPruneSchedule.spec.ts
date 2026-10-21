/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import type { SetupStatus } from "@/composable/setup/setup";
import HostPruneSchedule from "./HostPruneSchedule.vue";

const setup = vi.hoisted(() => ({
  status: undefined as unknown as import("vue").Ref<SetupStatus | undefined>,
  fetchStatus: vi.fn(),
  saveConfig: vi.fn(),
}));
vi.mock("@/composable/setup/setup", async (original) => ({
  ...(await original<typeof import("@/composable/setup/setup")>()),
  useSetup: () => setup,
}));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      reclaim: {
        "schedule-label": "Clean up automatically",
        "schedule-desc": "Never volumes.",
        "last-run": "Last run freed {size} {when}.",
        "last-run-failed": "{count} host failed. | {count} hosts failed.",
      },
      setup: {
        actions: { locked: "Set by {env}" },
        restart: { off: "Off" },
        update: { daily: "Daily", weekly: "Weekly on Sunday", "time-label": "Time", "when-label": "When" },
      },
      error: { "something-went-wrong": "Something went wrong" },
    },
  },
});

function status(extra: Partial<SetupStatus> = {}): SetupStatus {
  return {
    mode: "server",
    dataPersisted: true,
    canWrite: true,
    canWriteUpdates: true,
    locked: { authProvider: false, enableActions: false, enableShell: false },
    autoPrune: { mode: "off", time: "04:00", zone: "UTC" },
    ...extra,
  } as SetupStatus;
}

beforeEach(() => {
  setup.status = ref(status());
  setup.fetchStatus.mockReset();
  setup.saveConfig.mockReset().mockResolvedValue(undefined);
});

const mountIt = () => mount(HostPruneSchedule, { global: { plugins: [i18n] } });

describe("<HostPruneSchedule />", () => {
  test("turning it on saves weekly, and the selects save as they change", async () => {
    const wrapper = mountIt();
    expect(wrapper.find("select").exists()).toBe(false);

    await wrapper.find("input[type=checkbox]").setValue(true);
    await flushPromises();
    expect(setup.saveConfig).toHaveBeenLastCalledWith({ autoPrune: { mode: "weekly", time: "04:00" } });
    expect(wrapper.text()).toContain("UTC");

    await wrapper.find("select").setValue("daily");
    await flushPromises();
    expect(setup.saveConfig).toHaveBeenLastCalledWith({ autoPrune: { mode: "daily", time: "04:00" } });
  });

  test("the panel keeps its When row while off, disabled", () => {
    const wrapper = mount(HostPruneSchedule, { props: { panel: true }, global: { plugins: [i18n] } });
    const selects = wrapper.findAll("select");
    expect(selects).toHaveLength(2);
    expect(selects.every((s) => (s.element as HTMLSelectElement).disabled)).toBe(true);
  });

  test("puts the toggle back when the save fails", async () => {
    setup.saveConfig.mockRejectedValue(new Error("nope"));
    const wrapper = mountIt();
    await wrapper.find("input[type=checkbox]").setValue(true);
    await flushPromises();
    expect((wrapper.find("input[type=checkbox]").element as HTMLInputElement).checked).toBe(false);
    expect(wrapper.text()).toContain("Something went wrong");
  });

  test("is text for someone who can't change it", () => {
    setup.status.value = status({
      canWriteUpdates: false,
      autoPrune: { mode: "daily", time: "05:00", zone: "UTC" },
    });
    const wrapper = mountIt();
    expect(wrapper.find("select").exists()).toBe(false);
    expect(wrapper.find("input").exists()).toBe(false);
    expect(wrapper.text()).toContain("Daily");
    expect(wrapper.text()).toContain("05:00 UTC");
  });

  test("is text naming the variable when an env var fixes it", () => {
    setup.status = ref(
      status({
        locked: { authProvider: false, enableActions: false, enableShell: false, autoPruneMode: true },
        autoPrune: { mode: "daily", time: "04:00", zone: "UTC" },
      }),
    );
    const wrapper = mountIt();
    expect(wrapper.find("input").exists()).toBe(false);
    expect(wrapper.text()).toContain("Daily");
    expect(wrapper.text()).toContain("Set by DOZZLE_AUTO_CLEANUP / --auto-cleanup");
  });

  test("says what the last run did", () => {
    setup.status.value = status({
      autoPrune: {
        mode: "daily",
        time: "04:00",
        zone: "UTC",
        lastRun: { at: new Date().toISOString(), spaceReclaimed: 3 * 1024 ** 3, failed: 2 },
      },
    });
    const text = mountIt().text();
    expect(text).toContain("Last run freed 3 GB");
    expect(text).toContain("2 hosts failed.");
  });

  test("shows nothing before the status arrives, and asks for it", () => {
    setup.status = ref(undefined);
    expect(mountIt().html()).toBe("<!--v-if-->");
    expect(setup.fetchStatus).toHaveBeenCalled();
  });
});
