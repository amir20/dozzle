/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";
import { computed, nextTick, ref } from "vue";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { cloudUrl: "https://cloud.example", hosts: [] } }));
vi.mock("@/stores/settings", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/stores/settings")>()),
  collapseRail: ref(false),
}));
vi.mock("@/composable/ui/media", () => ({ isMobile: ref(false) }));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked: computed(() => true) }) }));
vi.mock("@/composable/cloud/cloudScope", () => ({ useCloudScope: () => ({ containers: computed(() => []) }) }));
vi.stubGlobal("trackUsage", vi.fn());

import { useCloudRail } from "./cloudRail";

describe("cloudRail", () => {
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

  // The strip is on every desktop page now, linked or not, so an embed that asked
  // for no chrome must not get it, nor lose its width to the offset.
  test("an embed with ?hideMenu gets no strip and no offset", () => {
    expect(useCloudRail().railOffset.value).toBe(48);
    history.replaceState(null, "", "/container/x?hideMenu");
    try {
      const rail = useCloudRail();
      expect(rail.mounted.value).toBe(false);
      expect(rail.collapsed.value).toBe(false);
      expect(rail.railOffset.value).toBe(0);
    } finally {
      history.replaceState(null, "", "/");
    }
  });
});
