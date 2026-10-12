/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";
import { ref, shallowRef, type Ref } from "vue";
import { DeployLogEntry, LoadMoreLogEntry, SimpleLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";
import type { ContainerUpdate } from "@/models/ContainerUpdate";
import type { Container } from "@/models/Container";
import { loadBetween } from "./loadBetween";
import { mergeByTime, olderCursor, reachedStartIn, useLogLoader } from "./logLoader";

vi.mock("./loadBetween", () => ({
  loadBetween: vi.fn(async () => ({ logs: [], signal: new AbortController().signal })),
}));

vi.mock("@/composable/cloud/alertMerger", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/composable/cloud/alertMerger")>()),
  useAlertMerger: () => ({ withAlerts: async (logs: unknown[]) => logs, decorateVisible: () => {} }),
}));

const T0 = Date.UTC(2026, 9, 2, 3, 0, 0);
const DAY = 24 * 3600_000;
const line = (ms: number) => new SimpleLogEntry("l", "a", ms, new Date(T0 + ms), "info", "stdout", "l");

describe("loadOlderLogs", () => {
  // The review's repro: a marker dated at yesterday's start above today's lines
  // used to be taken as the oldest line, so the load asked for lines from before
  // the container existed and scrolling up stopped loading.
  test("pages back from the oldest line, not from an update marker", async () => {
    const loader = new LoadMoreLogEntry(new Date(), async () => {});
    const marker = new DeployLogEntry({ newId: "a" } as ContainerUpdate, new Date(T0));
    const messages = shallowRef<LogEntry<LogMessage>[]>([loader, marker, line(DAY), line(DAY + 1_000)]);
    const loadedOlder = vi.fn();
    const { loadOlderLogs } = useLogLoader(
      messages,
      ref([{ id: "a", host: "h" } as Container]) as Ref<Container[]>,
      ref(new URLSearchParams()),
      ref(false),
      { loadedOlder },
    );

    await loadOlderLogs(loader);

    const [, , , to] = vi.mocked(loadBetween).mock.lastCall!;
    expect(to.getTime()).toBe(T0 + DAY);
    // Nothing older came back, so the container's start is reached.
    expect(loadedOlder).toHaveBeenCalledWith(new Set(["a"]));
  });
});

const of = (c: string, ms: number) => new SimpleLogEntry(`${c}${ms}`, c, ms, new Date(T0 + ms), "info", "stdout", "l");
const names = (logs: LogEntry<LogMessage>[]) => logs.map((l) => (l instanceof LoadMoreLogEntry ? "loader" : l.message));

describe("olderCursor", () => {
  const oldest = new Map([
    ["a", of("a", 5_000)],
    ["b", of("b", 1_000)],
    ["c", of("c", 9_000)],
  ]);

  test("is the latest oldest line among containers that have not reached their start", () => {
    expect(olderCursor(oldest, new Set())?.message).toBe("c9000");
    expect(olderCursor(oldest, new Set(["c"]))?.message).toBe("a5000");
  });

  test("is nothing once every container with a line on screen reached its start", () => {
    expect(olderCursor(oldest, new Set(["a", "b", "c"]))).toBeUndefined();
  });
});

describe("mergeByTime", () => {
  test("puts older lines between the lines on screen, not only above them", () => {
    const merged = mergeByTime([of("b", 1_000), of("a", 5_000)], [of("a", 500), of("a", 2_000)]);
    expect(names(merged)).toEqual(["a500", "b1000", "a2000", "a5000"]);
  });
});

describe("reachedStartIn", () => {
  test("names the containers that sent fewer lines than asked for, counted by container", () => {
    const containers = [{ id: "a" }, { id: "b" }, { id: "c" }] as Container[];
    expect(reachedStartIn(containers, [of("a", 1), of("a", 2), of("b", 3)], 2)).toEqual(new Set(["b", "c"]));
  });
});

describe("loadOlderLogs in a merged view", () => {
  test("one request from the cursor, dropping lines on screen and merging the rest by time", async () => {
    vi.mocked(loadBetween).mockClear();
    const containers = ref([
      { id: "a", host: "h" },
      { id: "b", host: "h" },
    ] as Container[]) as Ref<Container[]>;
    const loader = new LoadMoreLogEntry(new Date(), async () => {});
    // b is quiet: its oldest line on screen is far older than a's.
    const messages = shallowRef<LogEntry<LogMessage>[]>([loader, of("b", 1_000), of("a", 5_000), of("a", 6_000)]);
    const loadedOlder = vi.fn();
    const { loadOlderLogs } = useLogLoader(messages, containers, ref(new URLSearchParams()), ref(false), {
      loadedOlder,
    });
    // b's lines in a's window are all on screen already, so only a is asked, for
    // the whole page, and it sends the 100 it was asked for.
    const fromA = Array.from({ length: 100 }, (_, i) => of("a", 2_000 + i));
    vi.mocked(loadBetween).mockResolvedValueOnce({ logs: fromA, signal: new AbortController().signal });

    await loadOlderLogs(loader);

    expect(loadBetween).toHaveBeenCalledTimes(1);
    const [asked, , , to, options] = vi.mocked(loadBetween).mock.lastCall!;
    expect(asked.map((c) => c.id)).toEqual(["a"]);
    expect(to.getTime()).toBe(T0 + 5_000);
    expect(options).toMatchObject({ min: 100, lastSeenId: 5_000 });
    expect(names(messages.value)).toEqual(["loader", "b1000", ...fromA.map((l) => l.message), "a5000", "a6000"]);
    expect(loadedOlder).toHaveBeenLastCalledWith(new Set());

    // a's window now reaches past b's oldest line, so both are asked, from a's new
    // oldest line, and both come back short of the minimum: at their start.
    vi.mocked(loadBetween).mockResolvedValueOnce({ logs: [of("a", 100)], signal: new AbortController().signal });
    await loadOlderLogs(loader);
    const [again, , , nextTo, nextOptions] = vi.mocked(loadBetween).mock.lastCall!;
    expect(again.map((c) => c.id)).toEqual(["a", "b"]);
    expect(nextOptions).toMatchObject({ min: 50 });
    expect(nextTo.getTime()).toBe(T0 + 2_000);
    expect(names(messages.value).slice(0, 3)).toEqual(["loader", "a100", "b1000"]);
    expect(loadedOlder).toHaveBeenLastCalledWith(new Set(["a", "b"]));

    // Every container is at its start: there is nothing older to load.
    await loadOlderLogs(loader);
    expect(loadBetween).toHaveBeenCalledTimes(2);
  });
});

describe("loadOlderLogs while the stream flushes", () => {
  const twoContainers = () =>
    ref([
      { id: "a", host: "h" },
      { id: "b", host: "h" },
    ] as Container[]) as Ref<Container[]>;

  test("keeps live lines that arrive during the load, merging older lines into them", async () => {
    const loader = new LoadMoreLogEntry(new Date(), async () => {});
    const messages = shallowRef<LogEntry<LogMessage>[]>([loader, of("a", 5_000), of("b", 6_000)]);
    const { loadOlderLogs } = useLogLoader(messages, twoContainers(), ref(new URLSearchParams()), ref(false));
    vi.mocked(loadBetween).mockImplementationOnce(async () => {
      // A flush lands while the page is in flight.
      messages.value = [...messages.value, of("a", 7_000), of("b", 8_000)];
      return { logs: [of("a", 1_000), of("b", 2_000), of("a", 5_000)], signal: new AbortController().signal };
    });

    await loadOlderLogs(loader);

    expect(names(messages.value)).toEqual(["loader", "a1000", "b2000", "a5000", "b6000", "a7000", "b8000"]);
  });

  test("leaves a fresh window alone when the page lands after it opened", async () => {
    const loader = new LoadMoreLogEntry(new Date(), async () => {});
    const messages = shallowRef<LogEntry<LogMessage>[]>([loader, of("a", 5_000)]);
    const loadedOlder = vi.fn();
    const { loadOlderLogs } = useLogLoader(messages, twoContainers(), ref(new URLSearchParams()), ref(false), {
      loadedOlder,
    });
    const fresh = [new LoadMoreLogEntry(new Date(), async () => {}), of("b", 9_000)];
    vi.mocked(loadBetween).mockImplementationOnce(async () => {
      // A container joins the merged view: the stream reconnects under a new loader.
      messages.value = fresh;
      return { logs: [of("a", 1_000)], signal: new AbortController().signal };
    });

    await loadOlderLogs(loader);

    expect(messages.value).toBe(fresh);
    expect(loadedOlder).not.toHaveBeenCalled();
  });
});
