/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Container } from "@/models/Container";
import LogExportRows from "./LogExportRows.vue";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", enableDownload: true },
  withBase: (path: string) => path,
}));

const container = { id: "a", name: "a", host: "local" } as unknown as Container;

function mountRows(shortcuts?: boolean) {
  return mount(LogExportRows, {
    attachTo: document.body,
    props: { containers: [container], name: "a", ...(shortcuts === undefined ? {} : { shortcuts }) },
    global: { plugins: [createI18n({ legacy: false, missingWarn: false, fallbackWarn: false })] },
  });
}

const press = (key: string) => window.dispatchEvent(new KeyboardEvent("keydown", { key }));

describe("LogExportRows", () => {
  let submit: ReturnType<typeof vi.spyOn>;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    submit = vi.spyOn(HTMLFormElement.prototype, "submit").mockImplementation(() => {});
    fetchMock = vi.fn(async () => new Response("a log\n"));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  test("c copies and d downloads by default", () => {
    const wrapper = mountRows();
    press("c");
    press("d");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(submit).toHaveBeenCalledTimes(1);
    wrapper.unmount();
  });

  test("a pinned column leaves c and d to the main view", () => {
    const wrapper = mountRows(false);
    press("c");
    press("d");
    expect(fetchMock).not.toHaveBeenCalled();
    expect(submit).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});
