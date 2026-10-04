/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";
import { ref, shallowRef, type Ref } from "vue";
import { DeployLogEntry, LoadMoreLogEntry, SimpleLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";
import type { ContainerUpdate } from "@/models/ContainerUpdate";
import type { Container } from "@/models/Container";
import { loadBetween } from "./loadBetween";
import { useLogLoader } from "./logLoader";

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
