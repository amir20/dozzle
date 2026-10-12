/**
 * @vitest-environment jsdom
 */
import { beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";

const config = vi.hoisted(() => ({ hostname: "" }));
vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: config,
  withBase: (path: string) => path,
}));

async function load(hostname: string) {
  config.hostname = hostname;
  vi.resetModules();
  return await import("./title");
}

describe("document title", () => {
  beforeEach(() => {
    document.title = "";
  });

  test("is Dozzle with no subtitle and no hostname", async () => {
    await load("");
    await nextTick();
    expect(document.title).toBe("Dozzle");
  });

  test("names the hostname when one is configured", async () => {
    await load("box");
    await nextTick();
    expect(document.title).toBe("Dozzle @ box");
  });

  test("follows setTitle, and drops the prefix when cleared", async () => {
    const { setTitle } = await load("box");
    setTitle("nginx");
    await nextTick();
    expect(document.title).toBe("nginx - Dozzle @ box");

    setTitle("redis");
    await nextTick();
    expect(document.title).toBe("redis - Dozzle @ box");

    setTitle("");
    await nextTick();
    expect(document.title).toBe("Dozzle @ box");
  });
});
