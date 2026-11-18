/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, defineComponent, ref } from "vue";
import { createI18n } from "vue-i18n";

vi.mock("@/stores/config", () => ({ __esModule: true, default: {}, withBase: (p: string) => p }));
const scope = ref([{ host: "h1", id: "a1", name: "api" }]);
vi.mock("./cloudScope", () => ({ useCloudScope: () => ({ containers: computed(() => scope.value) }) }));
const toast = vi.hoisted(() => vi.fn());
vi.mock("@/composable/app/toast", () => ({ useToast: () => ({ showToast: toast }) }));

import {
  fetchFindings,
  findingMoment,
  findingsUrl,
  markFindingsSeen,
  resetSeenFindings,
  SEEN_LIMIT,
  useCloudFindings,
  useUnseenFindings,
  type CloudFinding,
} from "./cloudFindings";

const finding = (id: string, extra: Partial<CloudFinding> = {}): CloudFinding => ({
  id,
  severity: "high",
  status: "ongoing",
  locked: false,
  headline: `finding ${id}`,
  containers: [{ host: "h1", id: "a1", name: "api" }],
  daysOpen: 2,
  firstSeenAt: 1_700_000_000_000_000_000,
  lastSeenAt: 1_700_000_100_000_000_000,
  ...extra,
});

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

function mountFindings() {
  let api!: ReturnType<typeof useCloudFindings>;
  const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, messages: { en: {} } });
  mount(
    defineComponent({
      setup() {
        api = useCloudFindings("new");
        return () => null;
      },
    }),
    { global: { plugins: [i18n] } },
  );
  return api;
}

function mountUnseen() {
  let dot!: ReturnType<typeof useUnseenFindings>;
  mount(
    defineComponent({
      setup() {
        dot = useUnseenFindings();
        return () => null;
      },
    }),
  );
  return dot;
}

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
  resetSeenFindings();
  toast.mockReset();
  scope.value = [{ host: "h1", id: "a1", name: "api" }];
});

describe("findings scoping", () => {
  test("asks about each container on screen by host and id", () => {
    const url = findingsUrl(
      [
        { host: "h1", id: "a1" },
        { host: "h2", id: "b2" },
      ],
      "ignored",
    );
    expect(url).toBe("/api/cloud/findings?container=h1%2Fa1&container=h2%2Fb2&filter=ignored");
  });

  test("an empty scope asks nothing, since the instance's list lives in Cloud", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    expect(await fetchFindings([], "new")).toEqual({ findings: [], fixLocked: false, canIgnore: false });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  test("reloads when the scope changes", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(json({ findings: [finding("k1")] })));
    vi.stubGlobal("fetch", fetchMock);
    const api = mountFindings();
    await flushPromises();
    expect(api.findings.value.map((f) => f.id)).toEqual(["k1"]);

    scope.value = [{ host: "h1", id: "b2", name: "worker" }];
    await flushPromises();
    expect(fetchMock).toHaveBeenLastCalledWith("/api/cloud/findings?container=h1%2Fb2&filter=new");
  });

  test("Show me the lines opens the newest line, or the last scan without one", () => {
    const c = { host: "h1", id: "a1", name: "api" };
    expect(findingMoment(finding("k1", { logLastAt: 1_700_000_050_000_000_000 }), c).date.getTime()).toBe(
      1_700_000_050_000,
    );
    expect(findingMoment(finding("k1"), c)).toEqual({ containerId: "a1", date: new Date(1_700_000_100_000) });
  });
});

describe("ignoring a finding", () => {
  test("drops the row, posts the ignore and offers an undo", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ findings: [finding("k1"), finding("k2")] }))
      .mockResolvedValue(json({ finding: finding("k1") }));
    vi.stubGlobal("fetch", fetchMock);
    const api = mountFindings();
    await flushPromises();

    await api.ignore(api.findings.value[0]);
    expect(fetchMock).toHaveBeenLastCalledWith("/api/cloud/findings/k1/ignore", { method: "POST" });
    expect(api.findings.value.map((f) => f.id)).toEqual(["k2"]);

    // The undo takes the ignore back and puts the row where it was.
    toast.mock.calls[0][0].action.handler();
    await flushPromises();
    expect(fetchMock).toHaveBeenLastCalledWith("/api/cloud/findings/k1/ignore", { method: "DELETE" });
    expect(api.findings.value.map((f) => f.id)).toEqual(["k1", "k2"]);
  });

  test("an undo after the list moved on fetches it again instead of splicing a stale row", async () => {
    scope.value = [{ host: "h1", id: "a1", name: "api" }];
    // Answers by URL: earlier tests' components are still mounted and refetch
    // on the scope change too.
    const fetchMock = vi.fn(async (url: string) => {
      if (url.includes("/ignore")) return json({ finding: finding("k1") });
      return json({ findings: url.includes("a1") ? [finding("k1"), finding("k2")] : [finding("k1")] });
    });
    vi.stubGlobal("fetch", fetchMock);
    const api = mountFindings();
    await flushPromises();

    await api.ignore(api.findings.value[0]);
    scope.value = [{ host: "h1", id: "b2", name: "worker" }];
    await flushPromises();

    toast.mock.lastCall![0].action.handler();
    await flushPromises();
    expect(fetchMock).toHaveBeenCalledWith("/api/cloud/findings/k1/ignore", { method: "DELETE" });
    expect(fetchMock).toHaveBeenLastCalledWith("/api/cloud/findings?container=h1%2Fb2&filter=new");
    expect(api.findings.value.map((f) => f.id)).toEqual(["k1"]);
    scope.value = [{ host: "h1", id: "a1", name: "api" }];
  });

  test("puts the row back when Cloud refuses", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ findings: [finding("k1")] }))
      .mockResolvedValue(json({}, 403));
    vi.stubGlobal("fetch", fetchMock);
    const api = mountFindings();
    await flushPromises();

    await api.ignore(api.findings.value[0]);
    expect(api.findings.value.map((f) => f.id)).toEqual(["k1"]);
    expect(toast.mock.calls[0][0].type).toBe("error");
  });
});

describe("what the server decides", () => {
  test("canIgnore and truncated come from the response, and default to false", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(json({ findings: [finding("k1")], canIgnore: true, truncated: true })),
    );
    const api = mountFindings();
    await flushPromises();
    expect(api.canIgnore.value).toBe(true);
    expect(api.truncated.value).toBe(true);

    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json({ findings: [] })));
    await api.reload();
    expect(api.canIgnore.value).toBe(false);
    expect(api.truncated.value).toBe(false);
  });
});

describe("the findings dot", () => {
  const onApi = { host: "h1", id: "a1", name: "api" };
  const onWorker = { host: "h1", id: "b2", name: "worker" };

  test("reading one container's findings does not put out another's", async () => {
    // The panel opened on api marks api only.
    markFindingsSeen([finding("k1", { containers: [onApi], firstSeenAt: 200 })]);

    // worker has an older finding never read: still new to the reader.
    scope.value = [onWorker];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(json({ findings: [finding("k2", { containers: [onWorker], firstSeenAt: 100 })] })),
    );
    const dot = mountUnseen();
    await flushPromises();
    expect(dot.value).toBe(true);

    markFindingsSeen([finding("k2", { containers: [onWorker], firstSeenAt: 100 })]);
    expect(dot.value).toBe(false);
  });

  test("a finding already read stays out of the dot after a reload", async () => {
    markFindingsSeen([finding("k1", { containers: [onApi], firstSeenAt: 200 })]);
    expect(JSON.parse(localStorage.getItem("DOZZLE_LAST_SEEN_FINDING")!)).toEqual({ api: 200 });
    resetSeenFindings();

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(json({ findings: [finding("k1", { containers: [onApi], firstSeenAt: 200 })] })),
    );
    const dot = mountUnseen();
    await flushPromises();
    expect(dot.value).toBe(false);
  });

  test("the old single-number mark reads as nothing seen", () => {
    localStorage.setItem("DOZZLE_LAST_SEEN_FINDING", "123");
    resetSeenFindings();
    markFindingsSeen([finding("k1", { containers: [onApi], firstSeenAt: 5 })]);
    expect(JSON.parse(localStorage.getItem("DOZZLE_LAST_SEEN_FINDING")!)).toEqual({ api: 5 });
  });

  test("keeps the newest marks once past the limit", () => {
    markFindingsSeen(
      Array.from({ length: SEEN_LIMIT + 10 }, (_, i) =>
        finding(`k${i}`, { containers: [{ host: "h", id: `c${i}`, name: `c${i}` }], firstSeenAt: i + 1 }),
      ),
    );
    const stored = JSON.parse(localStorage.getItem("DOZZLE_LAST_SEEN_FINDING")!);
    expect(Object.keys(stored)).toHaveLength(SEEN_LIMIT);
    expect(stored.c0).toBeUndefined();
    expect(stored[`c${SEEN_LIMIT + 9}`]).toBe(SEEN_LIMIT + 10);
  });

  test("survives storage that throws", () => {
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(() => markFindingsSeen([finding("k1", { firstSeenAt: 9 })])).not.toThrow();
    setItem.mockRestore();
  });

  test("a scope left behind cannot answer for the one on screen", async () => {
    let answerOld!: (r: Response) => void;
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => new Promise<Response>((resolve) => (answerOld = resolve)))
      .mockResolvedValueOnce(json({ findings: [] }));
    vi.stubGlobal("fetch", fetchMock);
    scope.value = [onApi, onWorker];
    const dot = mountUnseen();

    // Narrowed to worker, whose answer (nothing new) lands first.
    scope.value = [onWorker];
    await flushPromises();
    answerOld(json({ findings: [finding("k1", { containers: [onWorker], firstSeenAt: 300 })] }));
    await flushPromises();
    expect(dot.value).toBe(false);
  });
});
