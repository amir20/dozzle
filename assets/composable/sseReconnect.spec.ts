/** @vitest-environment jsdom */
import { beforeEach, afterEach, describe, expect, test, vi } from "vitest";

// jsdom has no EventSource, and the readyState constants are all this needs
class FakeEventSource {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;
}
const { CONNECTING, OPEN, CLOSED } = FakeEventSource;

import { useSseReconnect } from "./sseReconnect";

describe("useSseReconnect", () => {
  let source: { readyState: number } | null;

  const setVisibility = (state: DocumentVisibilityState) => {
    Object.defineProperty(document, "visibilityState", { value: state, configurable: true });
  };

  const setOnline = (online: boolean) => {
    Object.defineProperty(navigator, "onLine", { value: online, configurable: true });
  };

  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0);
    global.EventSource = FakeEventSource as unknown as typeof EventSource;
    source = { readyState: CLOSED };
    setVisibility("visible");
    setOnline(true);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  test("retries a closed source with backoff", () => {
    const connect = vi.fn();
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    reconnect.onError();
    expect(connect).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1000);
    expect(connect).toHaveBeenCalledTimes(1);

    // second failure waits twice as long
    reconnect.onError();
    vi.advanceTimersByTime(1000);
    expect(connect).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1000);
    expect(connect).toHaveBeenCalledTimes(2);

    reconnect.dispose();
  });

  test("ignores an error while the browser is still retrying on its own", () => {
    const connect = vi.fn();
    source = { readyState: CONNECTING };
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    reconnect.onError();
    vi.advanceTimersByTime(60_000);
    expect(connect).not.toHaveBeenCalled();

    reconnect.dispose();
  });

  test("a successful open resets the backoff", () => {
    const connect = vi.fn();
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    reconnect.onError();
    vi.advanceTimersByTime(1000);
    reconnect.onError();
    reconnect.onOpen();

    vi.advanceTimersByTime(60_000);
    expect(connect).toHaveBeenCalledTimes(1);

    reconnect.onError();
    vi.advanceTimersByTime(1000);
    expect(connect).toHaveBeenCalledTimes(2);

    reconnect.dispose();
  });

  test("reconnects a closed source as soon as the tab is visible again", () => {
    const connect = vi.fn();
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    document.dispatchEvent(new Event("visibilitychange"));
    expect(connect).toHaveBeenCalledTimes(1);

    reconnect.dispose();
  });

  test("leaves a healthy source alone on wake", () => {
    const connect = vi.fn();
    source = { readyState: OPEN };
    const reconnect = useSseReconnect({
      connect,
      source: () => source as unknown as EventSource,
      staleAfter: 60_000,
    });

    reconnect.onOpen();
    vi.advanceTimersByTime(30_000);
    reconnect.onActivity();
    window.dispatchEvent(new Event("online"));
    expect(connect).not.toHaveBeenCalled();

    reconnect.dispose();
  });

  test("reconnects an open but silent source once the heartbeat is missed", () => {
    const connect = vi.fn();
    source = { readyState: OPEN };
    const reconnect = useSseReconnect({
      connect,
      source: () => source as unknown as EventSource,
      staleAfter: 60_000,
    });

    reconnect.onOpen();
    vi.advanceTimersByTime(61_000);
    window.dispatchEvent(new Event("online"));
    expect(connect).toHaveBeenCalledTimes(1);

    reconnect.dispose();
  });

  test("waits for the network before retrying", () => {
    const connect = vi.fn();
    setOnline(false);
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    document.dispatchEvent(new Event("visibilitychange"));
    expect(connect).not.toHaveBeenCalled();

    reconnect.dispose();
  });

  test("dispose stops a pending retry", () => {
    const connect = vi.fn();
    const reconnect = useSseReconnect({ connect, source: () => source as unknown as EventSource });

    reconnect.onError();
    reconnect.dispose();
    vi.advanceTimersByTime(60_000);
    expect(connect).not.toHaveBeenCalled();

    document.dispatchEvent(new Event("visibilitychange"));
    expect(connect).not.toHaveBeenCalled();
  });
});
