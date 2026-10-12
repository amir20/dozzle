/**
 * @vitest-environment jsdom
 */
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import type { Container } from "@/models/Container";
import { loadBetween } from "./loadBetween";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [] },
  withBase: (path: string) => path,
}));

describe("loadBetween", () => {
  afterEach(() => vi.unstubAllGlobals());

  test("posts every container and the window as a form body", async () => {
    const fetchMock = vi.fn(async (_url: string, _init: RequestInit) => new Response("", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await loadBetween(
      [{ id: "a", host: "h1" } as Container, { id: "b", host: "h2" } as Container],
      ref(new URLSearchParams("stdout=1")),
      new Date(0),
      new Date(1000),
      { min: 50, lastSeenId: 7 },
    );

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/containers/logs");
    expect(init.method).toBe("POST");
    expect(init.body).toBeInstanceOf(URLSearchParams);
    const body = init.body as URLSearchParams;
    expect(body.getAll("container")).toEqual(["h1/a", "h2/b"]);
    expect(body.get("format")).toBe("jsonl");
    expect(body.get("stdout")).toBe("1");
    expect(body.get("from")).toBe("1970-01-01T00:00:00.000Z");
    expect(body.get("to")).toBe("1970-01-01T00:00:01.000Z");
    expect(body.get("min")).toBe("50");
    expect(body.get("lastSeenId")).toBe("7");
  });

  test("throws the server's error instead of parsing it as lines", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("container not found\n", { status: 404 })),
    );

    await expect(
      loadBetween([{ id: "a", host: "h" } as Container], ref(new URLSearchParams()), new Date(0), new Date(1)),
    ).rejects.toThrow("404 container not found");
  });
});
