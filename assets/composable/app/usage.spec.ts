/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { createUsageBatcher } from "./usage";

function batcher(enabled = true) {
  const post = vi.fn<(body: string) => void>();
  let visible = true;
  const b = createUsageBatcher({ enabled: () => enabled, locale: () => "de", post, visible: () => visible });
  return { b, post, hide: () => (visible = false) };
}

describe("createUsageBatcher", () => {
  test("sends counts, minutes and the locale once", () => {
    const { b, post } = batcher();
    b.track("logs.sql");
    b.track("logs.sql");
    b.track("palette.open");
    b.tick();
    b.flush();

    expect(JSON.parse(post.mock.calls[0][0])).toEqual({
      counts: { "logs.sql": 2, "palette.open": 1 },
      activeMinutes: 1,
      locale: "de",
    });

    b.track("logs.search");
    b.flush();
    expect(JSON.parse(post.mock.calls[1][0])).toEqual({ counts: { "logs.search": 1 }, activeMinutes: 0 });
  });

  test("sends nothing when there is nothing new", () => {
    const { b, post } = batcher();
    b.flush();
    b.flush();
    expect(post).toHaveBeenCalledTimes(1);
  });

  test("a hidden page does not count as active", () => {
    const { b, post, hide } = batcher();
    hide();
    b.tick();
    b.track("pinned.open");
    b.flush();
    expect(JSON.parse(post.mock.calls[0][0]).activeMinutes).toBe(0);
  });

  test("does nothing with analytics off", () => {
    const { b, post } = batcher(false);
    b.track("logs.sql");
    b.tick();
    b.flush();
    expect(post).not.toHaveBeenCalled();
  });
});
