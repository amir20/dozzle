import { describe, expect, test } from "vitest";
import { computed, isReactive, nextTick, shallowRef } from "vue";
import { useSimpleRefHistory } from "./reactive";

describe("useSimpleRefHistory", () => {
  test("appends to the window and caps it at capacity", async () => {
    const source = shallowRef(0);
    const { history } = useSimpleRefHistory(source, { capacity: 3 });

    for (const n of [1, 2, 3, 4]) {
      source.value = n;
      await nextTick();
    }

    expect(history.value).toEqual([2, 3, 4]);
  });

  // The window is pushed into in place, so identity never changes and a reader is
  // only woken by the `triggerRef` inside. Drop it and every chart fed by this
  // freezes on its first frame with nothing else failing -- hence this test.
  test("wakes a reader on each new entry", async () => {
    const source = shallowRef(0);
    const { history } = useSimpleRefHistory(source, { capacity: 10 });

    let runs = 0;
    const latest = computed(() => {
      runs++;
      return history.value.at(-1);
    });

    expect(latest.value).toBeUndefined();
    expect(runs).toBe(1);

    source.value = 1;
    await nextTick();
    expect(latest.value).toBe(1);
    expect(runs).toBe(2);

    source.value = 2;
    await nextTick();
    expect(latest.value).toBe(2);
    expect(runs).toBe(3);
  });

  test("reset replaces the window and wakes readers", async () => {
    const source = shallowRef(0);
    const { history, reset } = useSimpleRefHistory(source, { capacity: 10 });

    const length = computed(() => history.value.length);
    expect(length.value).toBe(0);

    reset({ initial: [7, 8, 9] });
    expect(length.value).toBe(3);
    expect(history.value).toEqual([7, 8, 9]);
  });

  // Entries are read back once a second by the charts downstream, so they must not
  // be sitting behind a proxy. See the comment on the composable.
  test("keeps the window out of deep reactivity", async () => {
    const source = shallowRef({ value: 0 });
    const { history } = useSimpleRefHistory(source, { capacity: 10 });

    source.value = { value: 1 };
    await nextTick();

    expect(isReactive(history.value)).toBe(false);
    expect(isReactive(history.value.at(-1))).toBe(false);
  });
});
