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

import { fetchFindings, findingMoment, findingsUrl, useCloudFindings, type CloudFinding } from "./cloudFindings";

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

afterEach(() => {
  vi.unstubAllGlobals();
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
    expect(await fetchFindings([], "new")).toEqual({ findings: [], fixLocked: false });
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
