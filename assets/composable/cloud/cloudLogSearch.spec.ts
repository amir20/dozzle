/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { effectScope, nextTick, ref } from "vue";
import { useRouter } from "vue-router";
import { useCloudConfig } from "./cloudConfig";
import { useCloudLogSearch, useCloudSearchHandoff } from "./cloudLogSearch";

vi.mock("vue-router");
vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableCloud: true },
  withBase: (path: string) => path,
}));

const push = vi.fn();
const fetchMock = vi.fn();

beforeEach(() => {
  vi.useFakeTimers();
  vi.mocked(useRouter).mockReturnValue({ push } as unknown as ReturnType<typeof useRouter>);
  push.mockReset();
  fetchMock.mockReset().mockResolvedValue(new Response(JSON.stringify({ hits: [], hasMore: false })));
  vi.stubGlobal("fetch", fetchMock);
  useCloudConfig().cloudConfig.value = { linked: true, streamLogs: true } as never;
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("useCloudSearchHandoff", () => {
  test("carries find's regex intent to the results page", () => {
    const { open } = useCloudSearchHandoff();
    open("  err.*timeout  ", { regex: true });
    expect(push).toHaveBeenCalledWith({ path: "/cloud/search", query: { q: "err.*timeout", regex: "1" } });
    open("error");
    expect(push).toHaveBeenLastCalledWith({ path: "/cloud/search", query: { q: "error" } });
  });

  test("does nothing when cloud cannot search", () => {
    useCloudConfig().cloudConfig.value = { linked: true, streamLogs: false } as never;
    useCloudSearchHandoff().open("error", { regex: true });
    expect(push).not.toHaveBeenCalled();
  });
});

describe("useCloudLogSearch", () => {
  test("forwards regex to the backend and re-runs when it changes", async () => {
    const regex = ref(true);
    const scope = effectScope();
    scope.run(() => useCloudLogSearch(ref("err.*timeout"), regex));

    await vi.advanceTimersByTimeAsync(300);
    expect(fetchMock.mock.calls.at(-1)?.[0]).toContain("&regex=1");

    regex.value = false;
    await nextTick();
    await vi.advanceTimersByTimeAsync(300);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls.at(-1)?.[0]).not.toContain("regex");
    scope.stop();
  });
});
