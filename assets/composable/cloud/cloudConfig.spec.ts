import { afterEach, describe, expect, test, vi } from "vitest";
import { useCloudConfig } from "./cloudConfig";
import type { CloudStatus } from "@/types/notifications";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableCloud: true },
  withBase: (path: string) => path,
}));

const { cloudStatus, isPro, metricsLookbackHours } = useCloudConfig();

function setPlan(plan: Partial<CloudStatus["plan"]>) {
  cloudStatus.value = { plan: { name: "free", events_per_month: 0, retention_days: 0, ...plan } } as CloudStatus;
}

afterEach(() => (cloudStatus.value = null));

describe("plan entitlements", () => {
  test("renders what Cloud decided over the plan name", () => {
    setPlan({ name: "team", paid: true, metrics_lookback_hours: 720 });
    expect(isPro.value).toBe(true);
    expect(metricsLookbackHours.value).toBe(720);

    setPlan({ name: "pro", paid: false, metrics_lookback_hours: 24 });
    expect(isPro.value).toBe(false);
    expect(metricsLookbackHours.value).toBe(24);
  });

  test("an older Cloud falls back to the paid plan names, team included", () => {
    setPlan({ name: "Team" });
    expect(isPro.value).toBe(true);
    expect(metricsLookbackHours.value).toBe(720);

    setPlan({ name: "free" });
    expect(isPro.value).toBe(false);
    expect(metricsLookbackHours.value).toBe(24);
  });

  test("no status is not pro", () => {
    expect(isPro.value).toBe(false);
  });
});
