/**
 * @vitest-environment jsdom
 */
import { defineComponent } from "vue";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { beforeEach, afterEach, describe, expect, test, vi } from "vitest";

import { useCopy } from "./clipboard";
import { useToast } from "./toast";

const { toasts } = useToast();

function mountCopy() {
  let api: ReturnType<typeof useCopy>;
  const wrapper = mount(
    defineComponent({
      setup() {
        api = useCopy();
        return () => null;
      },
    }),
    { global: { plugins: [createI18n({})] } },
  );

  return { wrapper, api: api! };
}

function setSecureContext(value: boolean) {
  Object.defineProperty(window, "isSecureContext", { value, configurable: true });
}

describe("useCopy", () => {
  beforeEach(() => {
    toasts.value = [];
    setSecureContext(true);
    // @ts-expect-error jsdom has no clipboard; each test opts into one.
    delete navigator.clipboard;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("uses the async clipboard when it exists", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    const execCommand = vi.fn().mockReturnValue(true);
    document.execCommand = execCommand;

    const { api } = mountCopy();
    expect(await api.copy("hello")).toBe(true);

    expect(writeText).toHaveBeenCalledWith("hello");
    expect(execCommand).not.toHaveBeenCalled();
    expect(api.copied.value).toBe(true);
    expect(toasts.value.map(({ toast }) => toast.type)).toEqual(["info"]);
  });

  test("falls back to execCommand when there is no async clipboard", async () => {
    // What a plain http:// install looks like: navigator.clipboard is not there at all.
    setSecureContext(false);
    const execCommand = vi.fn().mockReturnValue(true);
    document.execCommand = execCommand;

    const { api } = mountCopy();
    expect(await api.copy("hello")).toBe(true);

    expect(execCommand).toHaveBeenCalledWith("copy");
    expect(api.copied.value).toBe(true);
    expect(toasts.value.map(({ toast }) => toast.type)).toEqual(["info"]);
  });

  test("explains itself instead of claiming success when the fallback fails too", async () => {
    setSecureContext(false);
    document.execCommand = vi.fn().mockReturnValue(false);

    const { api } = mountCopy();
    expect(await api.copy("hello")).toBe(false);

    expect(api.copied.value).toBe(false);
    const [{ toast }] = toasts.value;
    expect(toast.type).toBe("warning");
    expect(toast.title).toBe("error.copy-insecure");
    // The value is offered for manual copying, escaped because toasts render as HTML.
    expect(toast.message).toContain("error.copy-insecure-hint");
    expect(toast.message).toContain("hello");
  });

  test("falls back when the async clipboard rejects", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    const execCommand = vi.fn().mockReturnValue(true);
    document.execCommand = execCommand;

    const { api } = mountCopy();
    expect(await api.copy("hello")).toBe(true);

    expect(execCommand).toHaveBeenCalledWith("copy");
    expect(toasts.value.map(({ toast }) => toast.type)).toEqual(["info"]);
  });

  test("copyLazy only fetches the content once", async () => {
    setSecureContext(false);
    document.execCommand = vi.fn().mockReturnValue(true);
    const fetchText = vi.fn().mockResolvedValue("log line");

    const { api } = mountCopy();
    expect(await api.copyLazy(fetchText)).toBe(true);

    expect(fetchText).toHaveBeenCalledTimes(1);
  });

  test("copyLazy reports the fetch failure rather than a clipboard failure", async () => {
    const write = vi.fn().mockRejectedValue(new Error("blob rejected"));
    Object.defineProperty(navigator, "clipboard", { value: { write }, configurable: true });
    vi.stubGlobal(
      "ClipboardItem",
      class {
        constructor(public items: Record<string, unknown>) {}
      },
    );

    const { api } = mountCopy();
    await expect(api.copyLazy(() => Promise.reject(new Error("boom")))).rejects.toThrow("boom");

    expect(toasts.value).toHaveLength(0);
  });
});
