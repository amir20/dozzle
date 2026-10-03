import { describe, expect, test } from "vitest";
import { nextTick, ref } from "vue";
import { useStickyEntity } from "./stickyEntity";

describe("useStickyEntity", () => {
  test("keeps the last value while the key stays the same", async () => {
    const list = ref([{ name: "web" }]);
    const name = ref("web");
    const entity = useStickyEntity(
      () => list.value.find((e) => e.name === name.value),
      () => name.value,
    );
    expect(entity.value?.name).toBe("web");

    list.value = [];
    await nextTick();
    expect(entity.value?.name).toBe("web");
  });

  test("lets go when the key changes", async () => {
    const list = ref([{ name: "web" }]);
    const name = ref("web");
    const entity = useStickyEntity(
      () => list.value.find((e) => e.name === name.value),
      () => name.value,
    );

    list.value = [];
    name.value = "db";
    await nextTick();
    expect(entity.value).toBeUndefined();
  });

  test("is undefined for a key that never resolved", () => {
    const entity = useStickyEntity<{ name: string }>(
      () => undefined,
      () => "web",
    );
    expect(entity.value).toBeUndefined();
  });
});
