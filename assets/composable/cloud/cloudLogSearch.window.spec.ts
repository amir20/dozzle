/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { effectScope, ref } from "vue";
import { useCloudConfig } from "./cloudConfig";
import { useCloudLogSearch } from "./cloudLogSearch";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableCloud: true },
  withBase: (path: string) => path,
}));

const fetchMock = vi.fn();

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal("fetch", fetchMock);
  useCloudConfig().cloudConfig.value = { linked: true, streamLogs: true } as never;
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  fetchMock.mockReset();
});

function search() {
  const scope = effectScope();
  const result = scope.run(() => useCloudLogSearch(ref("timeout")))!;
  return { result, stop: () => scope.stop() };
}

describe("useCloudLogSearch window and total", () => {
  test("takes the window and the total from Cloud", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ hits: [], hasMore: true, windowStart: 1_700_000_000_000_000_000, total: 1234 })),
    );
    const { result, stop } = search();
    await vi.advanceTimersByTimeAsync(300);
    expect(result.windowStart.value).toBe(1_700_000_000_000_000_000);
    expect(result.total.value).toBe(1234);
    stop();
  });

  test("leaves them unset when Cloud does not send them", async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ hits: [], hasMore: true })));
    const { result, stop } = search();
    await vi.advanceTimersByTimeAsync(300);
    expect(result.windowStart.value).toBe(0);
    expect(result.total.value).toBeNull();
    stop();
  });
});
