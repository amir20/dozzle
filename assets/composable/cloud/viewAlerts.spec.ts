import { describe, expect, test, vi } from "vitest";
import { computed, ref } from "vue";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { cloudUrl: "https://cloud.example", hosts: [] } }));
vi.mock("@/stores/settings", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/stores/settings")>()),
  collapseCloudRail: ref(false),
}));
vi.mock("@/composable/ui/media", () => ({ isMobile: ref(false) }));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ linked: computed(() => true) }) }));
vi.stubGlobal("trackUsage", vi.fn());

// The home page: nothing on screen, so the panel asks about every container,
// cut at Cloud's limit.
vi.mock("@/composable/cloud/cloudScope", () => ({ useCloudScope: () => ({ containers: computed(() => []) }) }));
const instance = Array.from({ length: 101 }, (_, i) => ({
  host: "h",
  id: `c${String(i).padStart(3, "0")}`,
  name: `n${i}`,
}));
vi.mock("@/composable/cloud/cloudFindings", () => ({
  MAX_ASKED: 100,
  useFindingsScope: () => computed(() => instance),
}));

// One alert inside the cut, one on the container past it, both unseen.
const alerts = ref([
  { alertId: "a1", containerId: "c005", ts: 2 },
  { alertId: "a2", containerId: "c100", ts: 3 },
]);
vi.mock("@/composable/cloud/recentAlerts", async () => {
  const { computed, toValue } = await import("vue");
  return {
    useRecentAlerts: () => ({
      alerts,
      unseenIn: (ids: Set<string> | { value: Set<string> }) =>
        computed(() => alerts.value.some((a) => toValue(ids as never as Set<string>).has(a.containerId))),
    }),
  };
});

import { useViewAlerts } from "./cloudRail";

describe("useViewAlerts", () => {
  // The dot and the panel count the same containers: an alert on one past the
  // panel's cut would light a dot the panel has no row for.
  test("on a page with no containers, counts only the ones the panel asks about", () => {
    const { visible, unseen } = useViewAlerts();
    expect(visible.value.map((a) => a.alertId)).toEqual(["a1"]);
    expect(unseen.value).toBe(true);

    alerts.value = [{ alertId: "a2", containerId: "c100", ts: 3 }];
    expect(visible.value).toEqual([]);
    expect(unseen.value).toBe(false);
  });
});
