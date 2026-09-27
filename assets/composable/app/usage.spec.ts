/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { createUsageBatcher } from "./usage";

function batcher(enabled = true, accept = true) {
  const post = vi.fn<(body: string) => Promise<boolean>>(() => Promise.resolve(accept));
  let visible = true;
  const b = createUsageBatcher({ enabled: () => enabled, locale: () => "de", post, visible: () => visible });
  return { b, post, hide: () => (visible = false) };
}

describe("createUsageBatcher", () => {
  test("sends counts, minutes and the locale once", async () => {
    const { b, post } = batcher();
    b.track("logs.sql");
    b.track("logs.sql");
    b.track("palette.open");
    b.tick();
    await b.flush();

    expect(JSON.parse(post.mock.calls[0][0])).toEqual({
      counts: { "logs.sql": 2, "palette.open": 1 },
      activeMinutes: 1,
      locale: "de",
    });

    b.track("logs.search");
    await b.flush();
    expect(JSON.parse(post.mock.calls[1][0])).toEqual({ counts: { "logs.search": 1 }, activeMinutes: 0 });
  });

  test("sends nothing when there is nothing new", async () => {
    const { b, post } = batcher();
    await b.flush();
    await b.flush();
    expect(post).toHaveBeenCalledTimes(1);
  });

  test("a hidden page does not count as active", async () => {
    const { b, post, hide } = batcher();
    hide();
    b.tick();
    b.track("pinned.open");
    await b.flush();
    expect(JSON.parse(post.mock.calls[0][0]).activeMinutes).toBe(0);
  });

  test("does nothing with analytics off", async () => {
    const { b, post } = batcher(false);
    b.track("logs.sql");
    b.tick();
    await b.flush();
    expect(post).not.toHaveBeenCalled();
  });

  test("a failed report keeps its counts and the locale for the next one", async () => {
    let accept = false;
    const post = vi.fn<(body: string) => Promise<boolean>>(() => Promise.resolve(accept));
    const b = createUsageBatcher({ enabled: () => true, locale: () => "de", post, visible: () => true });

    b.track("logs.sql");
    b.tick();
    await b.flush();
    expect(post).toHaveBeenCalledTimes(1);

    b.track("logs.sql");
    accept = true;
    await b.flush();
    expect(JSON.parse(post.mock.calls[1][0])).toEqual({
      counts: { "logs.sql": 2 },
      activeMinutes: 1,
      locale: "de",
    });

    await b.flush();
    expect(post).toHaveBeenCalledTimes(2);
  });

  test("a report that throws is kept too", async () => {
    const post = vi.fn<(body: string) => Promise<boolean>>(() => Promise.reject(new Error("offline")));
    const b = createUsageBatcher({ enabled: () => true, locale: () => "de", post, visible: () => true });
    b.track("palette.open");
    await b.flush();
    post.mockImplementation(() => Promise.resolve(true));
    await b.flush();
    expect(JSON.parse(post.mock.calls[1][0]).counts).toEqual({ "palette.open": 1 });
  });
});
