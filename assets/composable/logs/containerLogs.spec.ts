/**
 * @vitest-environment jsdom
 */
import { defineComponent } from "vue";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Container } from "@/models/Container";

import { useContainerLogs } from "./containerLogs";
import { useToast } from "@/composable/app/toast";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableDownload: true },
  withBase: (path: string) => path,
}));

const { toasts } = useToast();

const container = (id: string, host = "local") => ({ id, name: id, host }) as unknown as Container;

function mountLogs(containers: Container[], name?: string) {
  let api: ReturnType<typeof useContainerLogs>;
  mount(
    defineComponent({
      setup() {
        api = useContainerLogs(containers, name);
        return () => null;
      },
    }),
    { global: { plugins: [createI18n({})] } },
  );
  return api!;
}

describe("useContainerLogs", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let writeText: ReturnType<typeof vi.fn<(value?: string) => boolean>>;

  beforeEach(() => {
    toasts.value = [];
    Object.defineProperty(window, "isSecureContext", { value: true, configurable: true });
    // No ClipboardItem in jsdom, so copyLazy waits for the text and runs execCommand.
    writeText = vi.fn().mockReturnValue(true);
    document.execCommand = vi.fn((command: string) => {
      if (command !== "copy") return false;
      writeText(document.querySelector("textarea")?.value);
      return true;
    });
    fetchMock = vi.fn(async () => new Response("── a ──\na log\n\n── b ──\nb log\n", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  test("copy is one request for every container, in view order, copied as returned", async () => {
    const { copyLogs } = mountLogs([container("a"), container("b", "agent-2")]);
    await copyLogs();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/containers/logs");
    expect(init.method).toBe("POST");
    expect(init.headers).toEqual({ Accept: "text/plain" });
    expect(String(init.body)).toBe("container=local%2Fa&container=agent-2%2Fb&stdout=1&stderr=1");
    expect(writeText).toHaveBeenCalledWith("── a ──\na log\n\n── b ──\nb log\n");
    expect(toasts.value.some(({ toast }) => toast.id === "copy-logs")).toBe(false);
  });

  test("download posts the same params as a form, asking for a named zip", () => {
    let submitted: HTMLFormElement | undefined;
    vi.spyOn(HTMLFormElement.prototype, "submit").mockImplementation(function (this: HTMLFormElement) {
      submitted = this;
    });
    const { downloadLogs } = mountLogs([container("a"), container("b")], "stack");
    downloadLogs();

    expect(submitted?.method).toBe("post");
    expect(submitted?.getAttribute("action")).toBe("/api/containers/logs");
    expect(submitted?.target).toBe("");
    expect(String(new URLSearchParams(new FormData(submitted!) as unknown as Record<string, string>))).toBe(
      "container=local%2Fa&container=local%2Fb&stdout=1&stderr=1&format=zip&name=stack",
    );
    expect(submitted?.isConnected).toBe(false);
  });

  test("a failed copy shows the server's reason and copies nothing", async () => {
    fetchMock.mockResolvedValueOnce(new Response("container db not found\n", { status: 404, statusText: "Not Found" }));
    const { copyLogs } = mountLogs([container("a")]);
    await copyLogs();

    expect(writeText).not.toHaveBeenCalled();
    const errors = toasts.value.filter(({ toast }) => toast.type === "error");
    expect(errors).toHaveLength(1);
    expect(errors[0].toast.message).toBe("container db not found");
  });

  test("an empty view copies nothing and asks for nothing", async () => {
    const { copyLogs } = mountLogs([]);
    await copyLogs();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
