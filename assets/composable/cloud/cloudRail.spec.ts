import { describe, expect, test, vi } from "vitest";
import { computed, nextTick, ref } from "vue";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { cloudUrl: "https://cloud.example", hosts: [] } }));
vi.mock("@/stores/settings", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/stores/settings")>()),
  collapseCloudRail: ref(false),
}));
vi.mock("@/composable/ui/media", () => ({ isMobile: ref(false) }));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked: computed(() => true) }) }));
const narrowBellWiden = vi.hoisted(() => vi.fn());
vi.mock("@/composable/cloud/cloudScope", () => ({ narrowBellWiden }));
vi.stubGlobal("trackUsage", vi.fn());

import { cloudPanelHref, useCloudRail } from "./cloudRail";

describe("cloudRail", () => {
  test("the account links to plan and billing, whatever is on screen", () => {
    expect(cloudPanelHref("account", { host: "h1", name: "api" })).toBe(
      "https://cloud.example/settings?from=dozzle_drawer_account",
    );
  });

  // The bell's widening lasts only as long as the drawer it opened.
  test("closing the drawer narrows a widening the bell made", async () => {
    const rail = useCloudRail();
    rail.openRail("alerts");
    await nextTick();
    expect(narrowBellWiden).not.toHaveBeenCalled();

    rail.openRail("findings");
    await nextTick();
    expect(narrowBellWiden).not.toHaveBeenCalled();

    rail.closeRail();
    await nextTick();
    expect(narrowBellWiden).toHaveBeenCalledOnce();
  });

  test("the account is never where a single Cloud entry lands", async () => {
    const rail = useCloudRail();
    rail.openRail("chat");
    await nextTick();
    rail.openRail("account");
    await nextTick();
    rail.closeRail();
    await nextTick();
    rail.openCloud(false);
    expect(rail.panel.value).toBe("chat");
    rail.closeRail();
  });
});
