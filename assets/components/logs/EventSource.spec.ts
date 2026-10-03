import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";
import { useSearchFilter } from "@/composable/logs/search";
import { settings } from "@/stores/settings";
// @ts-ignore
import EventSource, { sources } from "eventsourcemock";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { computed, nextTick } from "vue";
import { createI18n } from "vue-i18n";
import { createRouter, createWebHistory } from "vue-router";
import { default as Component } from "./EventSource.vue";
import SearchStatus from "./SearchStatus.vue";
import IndeterminateBar from "@/components/ui/IndeterminateBar.vue";
import ContainerEventLogItem from "./entries/ContainerEventLogItem.vue";
import type { TimeRange } from "@/composable/logs/timeRange";
import LogViewer from "./LogViewer.vue";
import { Container } from "@/models/Container";
import type { ContainerState } from "@/types/Container";
import { Level, type LogEntry } from "@/models/LogEntry";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", maxLogs: 400, authProvider: "none", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

/**
 * @vitest-environment jsdom
 */
describe("<ContainerEventSource />", () => {
  const search = useSearchFilter();

  beforeEach(() => {
    global.EventSource = EventSource;
    // @ts-ignore
    window.scrollTo = vi.fn();
    global.IntersectionObserver = class IntersectionObserver {
      observe = vi.fn();
      disconnect = vi.fn();
      unobserve = vi.fn();
      takeRecords = vi.fn();
      root = null;
      rootMargin = "";
      thresholds = [];
    } as any;
    vi.useFakeTimers();
    vi.setSystemTime(1560336942459);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  function createLogEventSource(
    {
      searchFilter = "",
      hourStyle = "auto",
      state = "running",
      timeRange = { kind: "live" },
    }: {
      searchFilter?: string | undefined;
      hourStyle?: "auto" | "24" | "12";
      state?: ContainerState;
      timeRange?: TimeRange;
    } = {
      hourStyle: "auto",
    },
  ) {
    settings.value.hourStyle = hourStyle;
    search.searchQueryFilter.value = searchFilter;
    if (searchFilter) {
      search.showSearch.value = true;
    }

    const router = createRouter({
      history: createWebHistory("/"),
      routes: [
        {
          path: "/",
          component: {
            template: "Test from createLogEventSource",
          },
        },
        {
          name: "/container/[id].time.[datetime]",
          path: "/container/:id/time/:datetime",
          component: {
            template: "Test from createLogEventSource",
          },
        },
      ],
    });

    return mount(Component, {
      global: {
        plugins: [
          router,
          createTestingPinia({
            createSpy: vi.fn,
            stubActions: false,
            initialState: {
              container: {
                containers: [
                  {
                    id: "abc",
                    image: "test:v123",
                    host: "localhost",
                    created: new Date(0),
                    finishedAt: new Date(0),
                    state: "running",
                  },
                ],
              },
            },
          }),
          createI18n({}),
        ],
        components: {
          LogViewer,
        },
        provide: {
          [scrollContextKey as symbol]: {
            paused: computed(() => false),
            loading: computed(() => false),
            progress: ref(1),
            available: ref(false),
            currentDate: ref(new Date()),
          },
          [loggingContextKey as symbol]: {
            containers: computed(() => [
              {
                id: "abc",
                image: "test:v123",
                host: "localhost",
                created: new Date(0),
                finishedAt: new Date(0),
                state,
              },
            ]),
            streamConfig: reactive({ stdout: true, stderr: true }),
            hasComplexLogs: ref(false),
            levels: new Set<Level>(["info"]),
            historical: ref(false),
            timeRange: ref(timeRange),
          },
        },
      },
      slots: {
        default: `
        <template #scoped="params"><LogViewer :messages="params.messages" :show-container-name="false" :visible-keys="[]" /></template>
        `,
      },
      props: {
        streamSource: useContainerStream,
        entity: new Container(
          "abc",
          new Date(), // created
          new Date(), // started
          new Date(), // finished
          "image",
          "name",
          "command",
          "localhost",
          {},
          state,
          0,
          0,
          [],
        ),
      },
    });
  }

  const sourceUrl = "/api/hosts/localhost/containers/abc/logs/stream?stdout=1&stderr=1&levels=info";

  test("renders loading correctly", async () => {
    const wrapper = createLogEventSource();
    expect(wrapper.find("ul.animate-pulse").exists()).toBe(true);
  });

  test("should connect to EventSource", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    expect(sources[sourceUrl].readyState).toBe(1);
    wrapper.unmount();
  });

  test("should close EventSource", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    wrapper.unmount();
    expect(sources[sourceUrl].readyState).toBe(2);
  });

  test("should parse messages", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    sources[sourceUrl].emitMessage({
      data: `{"ts":1560336942459, "m":"This is a message.", "id":1, "rm": "This is a message.", "c": "abc"}`,
    });

    vi.runAllTimers();
    await nextTick();

    // @ts-ignore
    const [message, _] = wrapper.vm.messages;
    expect(message).toMatchSnapshot();
  });

  test("keeps an overflowing opening burst when the next batch arrives", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    const emit = (id: number) =>
      sources[sourceUrl].emitMessage({
        data: `{"ts":${1560336942459 + id}, "m":"line ${id}", "id":${id}, "rm": "line ${id}", "c": "abc"}`,
      });

    for (let id = 1; id <= 500; id++) emit(id);
    await vi.advanceTimersByTimeAsync(200);
    // @ts-ignore
    const opening = wrapper.vm.messages.length;
    expect(opening).toBe(200);

    emit(501);
    await vi.advanceTimersByTimeAsync(1100);
    // @ts-ignore
    const messages: LogEntry<string>[] = wrapper.vm.messages;
    expect(messages).toHaveLength(opening + 1);
    expect(messages.at(-1)?.message).toBe("line 501");
  });

  test("keeps the view across a reconnect and drops the replayed tail", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    const emit = (id: number) =>
      sources[sourceUrl].emitMessage({
        data: `{"ts":${1560336942459 + id}, "m":"line ${id}", "id":${id}, "rm": "line ${id}", "c": "abc"}`,
      });

    for (let id = 1; id <= 5; id++) emit(id);
    await vi.advanceTimersByTimeAsync(200);
    // @ts-ignore
    const before: LogEntry<string>[] = wrapper.vm.messages;

    // the mock has no static readyState constants
    Object.assign(EventSource, { CONNECTING: 0, OPEN: 1, CLOSED: 2 });
    const dropped = sources[sourceUrl];
    dropped.readyState = 2;
    dropped.emitError();
    await vi.advanceTimersByTimeAsync(2000);
    expect(sources[sourceUrl]).not.toBe(dropped);

    sources[sourceUrl].emitOpen();
    for (let id = 3; id <= 7; id++) emit(id);
    await vi.advanceTimersByTimeAsync(1100);

    // @ts-ignore
    const after: LogEntry<string>[] = wrapper.vm.messages;
    expect(after.slice(0, before.length)).toEqual(before);
    expect(after.slice(before.length).map((m) => m.message)).toEqual(["line 6", "line 7"]);

    // one container's stdout and stderr can be stamped out of delivery order, and a
    // resume must not start sorting them
    sources[sourceUrl].emitMessage({
      data: `{"ts":${1560336942459 + 20}, "m":"line 20", "id":20, "rm": "line 20", "c": "abc"}`,
    });
    sources[sourceUrl].emitMessage({
      data: `{"ts":${1560336942459 + 15}, "m":"line 15", "id":15, "rm": "line 15", "c": "abc"}`,
    });
    await vi.advanceTimersByTimeAsync(1100);
    // @ts-ignore
    expect(wrapper.vm.messages.slice(-2).map((m: LogEntry<string>) => m.message)).toEqual(["line 20", "line 15"]);
    wrapper.unmount();
  });

  test("keeps lines buffered before the first flush when the stream drops", async () => {
    const wrapper = createLogEventSource();
    sources[sourceUrl].emitOpen();
    const emit = (id: number) =>
      sources[sourceUrl].emitMessage({
        data: `{"ts":${1560336942459 + id}, "m":"line ${id}", "id":${id}, "rm": "line ${id}", "c": "abc"}`,
      });

    for (let id = 1; id <= 3; id++) emit(id);
    // the browser retries on its own: same source, still CONNECTING
    sources[sourceUrl].readyState = 0;
    sources[sourceUrl].emitError();
    sources[sourceUrl].emitOpen();
    for (let id = 1; id <= 4; id++) emit(id);
    await vi.advanceTimersByTimeAsync(1100);

    // @ts-ignore
    const messages: LogEntry<string>[] = wrapper.vm.messages;
    expect(messages.slice(1).map((m) => m.message)).toEqual(["line 1", "line 2", "line 3", "line 4"]);
    wrapper.unmount();
  });

  test("puts matches from the gap after a resumed search, not above the view", async () => {
    const wrapper = createLogEventSource({ searchFilter: "line" });
    // the applied filter trails the typed one by a debounce
    await vi.advanceTimersByTimeAsync(1000);
    const url = Object.keys(sources).find((k) => k.includes("filter=line"))!;
    const event = (id: number) => ({
      ts: 1560336942459 + id,
      m: `line ${id}`,
      id,
      rm: `line ${id}`,
      c: "abc",
    });
    sources[url].emitOpen();
    sources[url].emit("logs-backfill", { data: JSON.stringify([event(1), event(2)]) });
    await vi.advanceTimersByTimeAsync(1100);

    sources[url].readyState = 0;
    sources[url].emitError();
    sources[url].emitOpen();
    sources[url].emit("logs-backfill", { data: JSON.stringify([event(0), event(2), event(3), event(4)]) });
    await vi.advanceTimersByTimeAsync(1100);

    // @ts-ignore
    const messages: LogEntry<string>[] = wrapper.vm.messages;
    expect(messages.map((m) => m.message)).toEqual(["line 0", "line 1", "line 2", "line 3", "line 4"]);
    search.searchQueryFilter.value = "";
    search.showSearch.value = false;
    wrapper.unmount();
  });

  describe("live bar", () => {
    test("lights up on an incoming batch and dims once the stream goes quiet", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"This is a message.", "id":1, "rm": "This is a message.", "c": "abc"}`,
      });

      // Past the 250ms buffer debounce, so the batch has flushed into messages.
      await vi.advanceTimersByTimeAsync(300);
      expect(wrapper.findComponent(IndeterminateBar).props("intensity")).toBe(1);

      await vi.advanceTimersByTimeAsync(2100);
      expect(wrapper.findComponent(IndeterminateBar).props("intensity")).toBe(0);
    });

    test("is hidden for a stopped container", () => {
      const wrapper = createLogEventSource({ state: "exited" });
      expect(wrapper.findComponent(IndeterminateBar).exists()).toBe(false);
    });
  });

  describe("container events", () => {
    const stopped = (time: string) => ({
      data: JSON.stringify({ actorId: "abc", name: "container-stopped", time }),
    });

    test("a stop replayed by a reconnect is not added twice", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emit("container-event", stopped("2026-10-03T10:00:00Z"));
      sources[sourceUrl].emit("container-event", stopped("2026-10-03T10:00:00Z"));
      await vi.advanceTimersByTimeAsync(300);
      expect(wrapper.findAllComponents(ContainerEventLogItem)).toHaveLength(1);
    });

    test("a stop before a since floor leaves the range empty", async () => {
      const since = new Date("2026-10-03T10:00:00Z");
      const wrapper = createLogEventSource({ timeRange: { kind: "since", since } });
      const url = Object.keys(sources).find((u) => u.includes("since="))!;
      sources[url].emitOpen();
      sources[url].emit("container-event", stopped("2026-10-03T08:00:00Z"));
      await vi.advanceTimersByTimeAsync(300);
      expect(wrapper.findAllComponents(ContainerEventLogItem)).toHaveLength(0);
    });
  });

  describe("search status", () => {
    test("shows no-logs when not searching and the stream is empty", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();

      await vi.advanceTimersByTimeAsync(3500);
      await nextTick();

      expect(wrapper.find('[data-testid="no-logs"]').exists()).toBe(true);
    });

    test("suppresses no-logs while a search is still running", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emit("search-status", {
        data: JSON.stringify({ scannedTo: "2026-06-01T14:31:00Z", matches: 0, done: false }),
      });

      vi.advanceTimersByTime(3000);
      await nextTick();

      expect(wrapper.find('[data-testid="no-logs"]').exists()).toBe(false);
      expect(wrapper.findComponent(SearchStatus).exists()).toBe(true);
    });
  });

  describe("render html correctly", () => {
    test("should render messages", async () => {
      const wrapper = createLogEventSource();
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"This is a message.", "id":1, "rm": "This is a message.", "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });

    test("should render dates with 12 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "12" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"foo bar", "id":1, "rm": "foo bar", "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });

    test("should render dates with 24 hour style", async () => {
      const wrapper = createLogEventSource({ hourStyle: "24" });
      sources[sourceUrl].emitOpen();
      sources[sourceUrl].emitMessage({
        data: `{"ts":1560336942459, "m":"foo bar", "id":1, "c": "abc"}`,
      });

      vi.runAllTimers();
      await nextTick();

      expect(wrapper.find("ul[data-logs]").html()).toMatchSnapshot();
    });
  });
});
