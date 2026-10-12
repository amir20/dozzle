/**
 * @vitest-environment jsdom
 */
import { defineComponent, reactive } from "vue";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Container } from "@/models/Container";
import type { Level } from "@/models/LogEntry";

import { useContainerLogs } from "./containerLogs";
import { useToast } from "@/composable/app/toast";
import { loggingContextKey } from "./logContext";
import { useSearchFilter } from "./search";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableDownload: true },
  withBase: (path: string) => path,
}));

const { toasts } = useToast();

const container = (id: string, host = "local") => ({ id, name: id, host }) as unknown as Container;

function mountLogs(containers: Container[], name?: string, levels?: Level[]) {
  let api: ReturnType<typeof useContainerLogs>;
  mount(
    defineComponent({
      setup() {
        api = useContainerLogs(containers, name);
        return () => null;
      },
    }),
    {
      global: {
        plugins: [createI18n({})],
        provide: levels
          ? {
              [loggingContextKey]: reactive({
                streamConfig: { stdout: true, stderr: true },
                levels: new Set(levels),
                timeRange: { kind: "live" },
              }),
            }
          : {},
      },
    },
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
    // Into a hidden frame, so an error response never replaces the app.
    const frame = document.querySelector<HTMLIFrameElement>(`iframe[name="${submitted?.target}"]`);
    expect(frame?.hidden).toBe(true);
    expect(String(new URLSearchParams(new FormData(submitted!) as unknown as Record<string, string>))).toBe(
      "container=local%2Fa&container=local%2Fb&stdout=1&stderr=1&format=zip&name=stack",
    );
    expect(submitted?.isConnected).toBe(false);
  });

  test("a failed download shows the server's reason from the frame and keeps the frame", () => {
    vi.spyOn(HTMLFormElement.prototype, "submit").mockImplementation(() => {});
    const { downloadLogs } = mountLogs([container("a")]);
    downloadLogs();
    const frame = document.querySelector<HTMLIFrameElement>('iframe[name="dozzle-download"]')!;
    frame.contentDocument!.body.textContent = "no containers found\n";
    frame.dispatchEvent(new Event("load"));

    const errors = toasts.value.filter(({ toast }) => toast.type === "error");
    expect(errors).toHaveLength(1);
    expect(errors[0].toast.message).toBe("no containers found");

    downloadLogs();
    expect(document.querySelectorAll('iframe[name="dozzle-download"]')).toHaveLength(1);
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

  test("a level filter is sent with each level kept, and counts as filtered", async () => {
    const { copyLogs, isFiltered } = mountLogs([container("a")], undefined, ["error", "warn"]);
    expect(isFiltered.value).toBe(true);
    await copyLogs();
    expect(String(fetchMock.mock.calls[0][1].body)).toBe(
      "container=local%2Fa&stdout=1&stderr=1&levels=error&levels=warn",
    );
  });

  test("a search is sent with its inverse flag, after the stream params", async () => {
    const { showSearch, appliedSearchFilter, inverseFilter } = useSearchFilter();
    const { copyLogs, isFiltered } = mountLogs([container("a")]);
    showSearch.value = true;
    appliedSearchFilter.value = "timeout";
    inverseFilter.value = true;
    try {
      expect(isFiltered.value).toBe(true);
      const copied = copyLogs();
      expect(String(fetchMock.mock.calls[0][1].body)).toBe(
        "container=local%2Fa&stdout=1&stderr=1&filter=timeout&inverse=true",
      );
      await copied;
    } finally {
      inverseFilter.value = false;
      showSearch.value = false;
      appliedSearchFilter.value = "";
    }
  });

  test("with every level cleared, copy and download ask for nothing", async () => {
    const submit = vi.spyOn(HTMLFormElement.prototype, "submit").mockImplementation(() => {});
    const { copyLogs, downloadLogs, isFiltered } = mountLogs([container("a")], undefined, []);
    expect(isFiltered.value).toBe(true);
    await copyLogs();
    downloadLogs();
    expect(fetchMock).not.toHaveBeenCalled();
    expect(submit).not.toHaveBeenCalled();
  });
});
