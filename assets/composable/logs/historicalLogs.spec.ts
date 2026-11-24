/**
 * @vitest-environment jsdom
 */
import { defineComponent, ref, shallowRef, type Ref } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { afterEach, describe, expect, test, vi } from "vitest";
import { HistoricalContainer, type Container } from "@/models/Container";
import { LoadMoreLogEntry, SimpleLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";
import { loadBetween } from "./loadBetween";
import { useHistoricalContainerLog } from "./historicalLogs";

vi.mock("vue-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("vue-router")>()),
  useRoute: () => ({ query: {} }),
  useRouter: () => ({}),
}));

vi.mock("./loadBetween", () => ({
  loadBetween: vi.fn(async () => ({ logs: [], signal: new AbortController().signal })),
}));

// A Cloud-linked instance: decorating a page waits on a request, held here until
// the test lets it go.
let held: (() => void)[] = [];
let hold = false;
vi.mock("@/composable/cloud/alertMerger", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/composable/cloud/alertMerger")>()),
  useAlertMerger: () => ({
    withAlerts: (logs: unknown[]) =>
      hold ? new Promise((resolve) => held.push(() => resolve(logs))) : Promise.resolve(logs),
    decorateVisible: () => {},
    alertsAvailable: ref(true),
  }),
}));

const T0 = Date.UTC(2026, 9, 2, 3, 0, 0);
const of = (c: string, ms: number) => new SimpleLogEntry(`${c}${ms}`, c, ms, new Date(T0 + ms), "info", "stdout", "l");
const names = (logs: LogEntry<LogMessage>[]) =>
  logs.map((l) => (l instanceof LoadMoreLogEntry ? "loader" : (l.message as string)));
const page = (logs: LogEntry<LogMessage>[]) => ({ logs, signal: new AbortController().signal });
const container = (id: string) => ({ id, name: id, host: "h" }) as unknown as Container;

async function open(target: Ref<HistoricalContainer>, before: LogEntry<LogMessage>[], after: LogEntry<LogMessage>[]) {
  vi.mocked(loadBetween).mockResolvedValueOnce(page(before)).mockResolvedValueOnce(page(after));
  let api!: ReturnType<typeof useHistoricalContainerLog>;
  mount(
    defineComponent({
      setup() {
        api = useHistoricalContainerLog(target);
        return () => null;
      },
    }),
    { global: { plugins: [createI18n({ legacy: false })] } },
  );
  await flushPromises();
  return api.messages;
}

afterEach(() => {
  hold = false;
  held = [];
  vi.mocked(loadBetween).mockReset();
  vi.useRealTimers();
});

describe("useHistoricalContainerLog", () => {
  test("keeps both ends when loads at the top and the bottom land together", async () => {
    const messages = await open(
      shallowRef(new HistoricalContainer(container("a"), new Date(T0 + 5_000))),
      [of("a", 5_000)],
      [of("a", 6_000)],
    );
    const older = messages.value[0] as LoadMoreLogEntry;
    const newer = messages.value.at(-1) as LoadMoreLogEntry;
    hold = true;
    vi.mocked(loadBetween)
      .mockResolvedValueOnce(page([of("a", 1_000)]))
      .mockResolvedValueOnce(page([of("a", 9_000)]));

    const olderDone = older.loadMore();
    const newerDone = newer.loadMore();
    await flushPromises();
    held[1](); // the bottom lands first
    await newerDone;
    held[0]();
    await olderDone;

    expect(names(messages.value)).toEqual(["loader", "a1000", "a5000", "a6000", "a9000", "loader"]);
  });

  test("leaves a window opened while a page was loading alone", async () => {
    const target = shallowRef(new HistoricalContainer(container("a"), new Date(T0 + 5_000)));
    const messages = await open(target, [of("a", 5_000)], []);
    const older = messages.value[0] as LoadMoreLogEntry;
    hold = true;
    vi.mocked(loadBetween).mockResolvedValueOnce(page([of("a", 1_000)]));
    const olderDone = older.loadMore();
    await flushPromises();

    // Another container opens; its window lands before the old page does.
    hold = false;
    vi.mocked(loadBetween)
      .mockResolvedValueOnce(page([of("b", 7_000)]))
      .mockResolvedValueOnce(page([]));
    target.value = new HistoricalContainer(container("b"), new Date(T0 + 7_000));
    await flushPromises();
    held[0]();
    await olderDone;

    expect(names(messages.value)).toEqual(["loader", "b7000", "loader"]);
  });

  test("gives the top and bottom loaders their own keys", async () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date(T0));
    const messages = await open(shallowRef(new HistoricalContainer(container("a"), new Date(T0))), [of("a", 0)], []);
    expect(messages.value[0].id).not.toBe(messages.value.at(-1)!.id);
  });
});
