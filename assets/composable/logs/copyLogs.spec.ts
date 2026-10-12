/**
 * @vitest-environment jsdom
 */
import { defineComponent } from "vue";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Container } from "@/models/Container";

import { COPY_CONCURRENCY, formatCopyBlocks, mapLimit, useCopyLogs } from "./copyLogs";
import { useToast } from "@/composable/app/toast";

const { toasts } = useToast();

const container = (id: string, name = id) => ({ id, name, host: "local" }) as unknown as Container;

function mountCopy(containers: Container[]) {
  let api: ReturnType<typeof useCopyLogs>;
  mount(
    defineComponent({
      setup() {
        api = useCopyLogs(containers);
        return () => null;
      },
    }),
    { global: { plugins: [createI18n({})] } },
  );
  return api!;
}

describe("formatCopyBlocks", () => {
  test("a single container is returned untouched, with no header", () => {
    const text = "line 1\nline 2\n";
    expect(formatCopyBlocks([{ name: "web", text }])).toBe(text);
  });

  test("several containers get a header each and a blank line between", () => {
    expect(
      formatCopyBlocks([
        { name: "web", text: "a\nb\n" },
        { name: "db", text: "c" },
        { name: "cache", text: "" },
      ]),
    ).toBe("── web ──\na\nb\n\n── db ──\nc\n\n── cache ──\n");
  });
});

describe("mapLimit", () => {
  test("never has more than the limit in flight and keeps input order", async () => {
    let inFlight = 0;
    let peak = 0;
    const items = Array.from({ length: 10 }, (_, i) => i);
    const result = await mapLimit(items, COPY_CONCURRENCY, async (i) => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await new Promise((r) => setTimeout(r, 10 - i));
      inFlight--;
      return i * 2;
    });
    expect(peak).toBe(COPY_CONCURRENCY);
    expect(result).toEqual(items.map((i) => i * 2));
  });
});

describe("useCopyLogs", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let writeText: ReturnType<typeof vi.fn<(value?: string) => boolean>>;

  beforeEach(() => {
    toasts.value = [];
    Object.defineProperty(window, "isSecureContext", { value: true, configurable: true });
    // No ClipboardItem in jsdom, so copyLazy waits for the text and runs execCommand.
    writeText = vi.fn().mockReturnValue(true);
    document.execCommand = vi.fn((command: string) => {
      if (command !== "copy") return false;
      const textarea = document.querySelector("textarea");
      writeText(textarea?.value);
      return true;
    });
    fetchMock = vi.fn(async (url: string) => {
      const id = /containers\/([^/]+)\/logs/.exec(url)![1];
      if (id === "bad") return new Response("", { status: 500, statusText: "Internal Server Error" });
      return new Response(`${id} log\n`, { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  test("a single container copies the endpoint's text as is", async () => {
    const { copyLogs } = mountCopy([container("one")]);
    await copyLogs();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/api/hosts/local/containers/one/logs?stdout=1&stderr=1&everything=1");
    expect(init).toEqual({ headers: { Accept: "text/plain" } });
    expect(writeText).toHaveBeenCalledWith("one log\n");
  });

  test("several containers copy one block each, in view order", async () => {
    const { copyLogs } = mountCopy([container("a", "web"), container("b", "db")]);
    await copyLogs();

    expect(writeText).toHaveBeenCalledWith("── web ──\na log\n\n── db ──\nb log\n");
    expect(toasts.value.some(({ toast }) => toast.id === "copy-logs")).toBe(false);
  });

  test("one container failing fails the copy and names it", async () => {
    const { copyLogs } = mountCopy([container("a", "web"), container("bad", "db")]);
    await copyLogs();

    expect(writeText).not.toHaveBeenCalled();
    const errors = toasts.value.filter(({ toast }) => toast.type === "error");
    expect(errors).toHaveLength(1);
    expect(errors[0].toast.message).toBe("db: Internal Server Error");
    expect(toasts.value.some(({ toast }) => toast.id === "copy-logs")).toBe(false);
  });
});
