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
