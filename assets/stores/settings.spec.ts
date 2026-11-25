import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "" },
  withBase: (path: string) => path,
}));

// A plain ref stands in for useStorage, so the test can do what a write from another
// tab does: replace settings.value with a fresh object.
vi.mock("@/composable/app/profileStorage", async () => {
  const { ref } = await import("vue");
  return { useProfileStorage: (_: string, defaults: object) => ref({ ...defaults }) };
});

const { settings, compact } = await import("./settings");

describe("settings refs", () => {
  test("follow settings after another tab replaces the object", () => {
    settings.value = { ...settings.value, compact: false };
    compact.value = true;
    expect(settings.value.compact).toBe(true);
  });
});
