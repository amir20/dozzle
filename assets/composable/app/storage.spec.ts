/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { shallowRef } from "vue";
import { Container } from "@/models/Container";
import { persistentVisibleKeysForContainer, visibleKeysForContainer } from "./storage";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

function makeContainer(image: string): Container {
  return new Container(
    "id-1",
    new Date(),
    new Date(),
    new Date(),
    image,
    "name",
    "command",
    "localhost",
    {},
    "running",
    0,
    0,
    [],
  );
}

describe("persistentVisibleKeysForContainer", () => {
  test("persists per storage key", () => {
    const container = shallowRef<Container | undefined>(makeContainer("image-a"));
    const visibleKeys = persistentVisibleKeysForContainer(container);
    const keys = new Map([[["secret"], false]]);

    visibleKeys.value = keys;

    expect(visibleKeys.value).toEqual(keys);
    expect(visibleKeysForContainer(container.value)).toEqual(keys);
    expect(visibleKeysForContainer(makeContainer("image-b")).size).toBe(0);
  });

  test("stays usable when the container is gone", () => {
    // A replica can be destroyed while its log lines are still on screen.
    const container = shallowRef<Container | undefined>(undefined);
    const visibleKeys = persistentVisibleKeysForContainer(container);

    expect(visibleKeys.value.size).toBe(0);
    expect(() => (visibleKeys.value = new Map([[["secret"], false]]))).not.toThrow();
    expect(visibleKeys.value.size).toBe(0);
  });
});
