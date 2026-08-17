/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";
import { effectScope, shallowRef, ref } from "vue";
import { useAlertMerger, isStreamLog } from "./alertMerger";
import type { CloudAlert } from "./cloudAlerts";
import {
  AlertLogEntry,
  CloudEventLogEntry,
  LoadMoreLogEntry,
  SimpleLogEntry,
  SkippedLogsEntry,
  type LogEntry,
  type LogMessage,
} from "@/models/LogEntry";
import type { Container } from "@/models/Container";

// Reactive so tests can flip the instance between linked and unlinked, which is
// what drives the poll and the boot-race decoration.
type CloudLink = { linked: boolean };
const holder = vi.hoisted(() => ({ cloudConfig: null as ReturnType<typeof import("vue").ref<CloudLink>> | null }));
vi.mock("@/composable/cloudConfig", async () => {
  const { ref: vueRef } = await import("vue");
  holder.cloudConfig = vueRef<CloudLink>({ linked: true });
  return { useCloudConfig: () => ({ cloudConfig: holder.cloudConfig }) };
});
const setLinked = (linked: boolean) => (holder.cloudConfig!.value = { linked });

const ns = (n: number) => n * 1_000_000;

function log(id: number, at: number): LogEntry<LogMessage> {
  return new SimpleLogEntry(`line ${id}`, "abc", id, new Date(at), "info", "stdout", `line ${id}`);
}

function alert(overrides: Partial<CloudAlert> = {}): CloudAlert {
  return {
    alertId: "a1",
    containerId: "abc",
    hostId: "h",
    ts: ns(100),
    headline: "Pool exhausted",
    level: "error",
    eventCount: 3,
    createdAt: ns(100),
    isOrigin: true,
    ...overrides,
  };
}

const shapeOf = (entries: LogEntry<LogMessage>[]) =>
  entries.map((e) => {
    if (e instanceof LoadMoreLogEntry) return "loader";
    if (e instanceof AlertLogEntry) return `alert:${e.alert.alertId}`;
    return `log:${e.id}`;
  });

function withMerger(
  messages: ReturnType<typeof shallowRef<LogEntry<LogMessage>[]>>,
  fn: (merger: ReturnType<typeof useAlertMerger>) => Promise<void>,
) {
  const scope = effectScope();
  const containers = shallowRef<Container[]>([{ id: "abc" } as unknown as Container]);
  const params = ref(new URLSearchParams());
  const merger = scope.run(() => useAlertMerger(messages as any, containers, params))!;
  return fn(merger).finally(() => scope.stop());
}

function respondWith(hits: CloudAlert[]) {
  global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ hits }) });
}

describe("isStreamLog", () => {
  test("accepts log lines and rejects the synthetic rows", () => {
    expect(isStreamLog(log(1, 100))).toBe(true);
    expect(isStreamLog(new LoadMoreLogEntry(new Date(), async () => {}))).toBe(false);
    expect(isStreamLog(new AlertLogEntry(alert(), new Date(100)))).toBe(false);
    expect(isStreamLog(new CloudEventLogEntry({ ts: ns(1), containerId: "abc", suppressed: true }, new Date(1)))).toBe(
      false,
    );
    expect(isStreamLog(new SkippedLogsEntry(new Date(), 1, log(1, 1) as any, log(2, 2) as any, async () => {}))).toBe(
      false,
    );
  });
});

describe("useAlertMerger", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setLinked(true);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  test("withAlerts splices an alert after the line that triggered it", async () => {
    respondWith([alert({ logId: 2, ts: ns(200) })]);
    const messages = shallowRef<LogEntry<LogMessage>[]>([]);
    await withMerger(messages, async ({ withAlerts }) => {
      const merged = await withAlerts([log(1, 100), log(2, 200), log(3, 300)]);
      expect(shapeOf(merged)).toEqual(["log:1", "log:2", "alert:a1", "log:3"]);
    });
  });

  test("decorateVisible keeps load-more rows at both ends", async () => {
    // A time-anchored alert older than every line on screen: without holding the
    // loaders out of the merge it would sort ahead of the head one.
    respondWith([alert({ ts: ns(50) })]);
    const messages = shallowRef<LogEntry<LogMessage>[]>([
      new LoadMoreLogEntry(new Date(), async () => {}),
      log(1, 100),
      log(2, 200),
      new LoadMoreLogEntry(new Date(), async () => {}, false),
    ]);

    await withMerger(messages, async ({ decorateVisible }) => {
      decorateVisible();
      await vi.advanceTimersByTimeAsync(500);
      expect(shapeOf(messages.value)).toEqual(["loader", "alert:a1", "log:1", "log:2", "loader"]);
    });
  });

  test("polls the visible window so an alert can land while the view is open", async () => {
    respondWith([]);
    const messages = shallowRef<LogEntry<LogMessage>[]>([log(1, 100), log(2, 200)]);

    await withMerger(messages, async ({ decorateVisible }) => {
      decorateVisible();
      await vi.advanceTimersByTimeAsync(500);
      expect(shapeOf(messages.value)).toEqual(["log:1", "log:2"]);

      respondWith([alert({ logId: 1, ts: ns(100) })]);
      await vi.advanceTimersByTimeAsync(15_000);
      expect(shapeOf(messages.value)).toEqual(["log:1", "alert:a1", "log:2"]);
    });
  });

  test("asks cloud for nothing while the instance is unlinked", async () => {
    respondWith([alert({ logId: 1, ts: ns(100) })]);
    setLinked(false);
    const messages = shallowRef<LogEntry<LogMessage>[]>([log(1, 100), log(2, 200)]);

    await withMerger(messages, async ({ decorateVisible }) => {
      decorateVisible();
      await vi.advanceTimersByTimeAsync(60_000);
      expect(global.fetch).not.toHaveBeenCalled();
      expect(shapeOf(messages.value)).toEqual(["log:1", "log:2"]);
    });
  });

  test("decorates the open window as soon as cloud links", async () => {
    // The boot race: cloudConfig is fetched asynchronously, so the first window
    // is usually assembled before the instance looks linked.
    respondWith([alert({ logId: 1, ts: ns(100) })]);
    setLinked(false);
    const messages = shallowRef<LogEntry<LogMessage>[]>([log(1, 100), log(2, 200)]);

    await withMerger(messages, async () => {
      await vi.advanceTimersByTimeAsync(500);
      expect(shapeOf(messages.value)).toEqual(["log:1", "log:2"]);

      setLinked(true);
      await vi.advanceTimersByTimeAsync(500);
      expect(shapeOf(messages.value)).toEqual(["log:1", "alert:a1", "log:2"]);
    });
  });

  test("drops a response whose window was replaced while it was in flight", async () => {
    let release: (res: unknown) => void = () => {};
    global.fetch = vi.fn().mockImplementation(() => new Promise((resolve) => (release = resolve)));
    const messages = shallowRef<LogEntry<LogMessage>[]>([log(1, 100), log(2, 200)]);

    await withMerger(messages, async ({ decorateVisible }) => {
      decorateVisible();
      await vi.advanceTimersByTimeAsync(500);

      // Container switch (or a live flush) lands before cloud answers.
      const replaced = [log(3, 300)];
      messages.value = replaced;
      release({ ok: true, json: async () => ({ hits: [alert({ logId: 1, ts: ns(100) })] }) });
      await vi.advanceTimersByTimeAsync(0);

      expect(messages.value).toBe(replaced);

      // And the dropped alert is not marked as placed, so the next pass draws it.
      respondWith([alert({ logId: 1, ts: ns(100) })]);
      await vi.advanceTimersByTimeAsync(15_000);
      expect(shapeOf(messages.value)).toEqual(["alert:a1", "log:3"]);
    });
  });

  test("does not place the same alert twice across overlapping windows", async () => {
    respondWith([alert({ logId: 2, ts: ns(200) })]);
    const messages = shallowRef<LogEntry<LogMessage>[]>([]);
    await withMerger(messages, async ({ withAlerts }) => {
      const first = await withAlerts([log(1, 100), log(2, 200)]);
      expect(shapeOf(first)).toEqual(["log:1", "log:2", "alert:a1"]);
      const second = await withAlerts([log(2, 200), log(3, 300)]);
      expect(shapeOf(second)).toEqual(["log:2", "log:3"]);
    });
  });
});
