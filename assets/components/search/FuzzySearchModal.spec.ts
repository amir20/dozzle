import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";

import FuzzySearchModal from "./FuzzySearchModal.vue";

import { Container } from "@/models/Container";
import { lightTheme } from "@/stores/settings";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { router } from "@/modules/router";

import EventSource from "@/testing/eventSourceMock";
import { useCloudConfig } from "@/composable/cloud/cloudConfig";

vi.mock("vue-router");

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }], enableActions: true },
  withBase: (path: string) => path,
}));

function createFuzzySearchModal(extra: Container[] = []) {
  global.EventSource = EventSource;
  const wrapper = mount(FuzzySearchModal, {
    global: {
      plugins: [
        createI18n({}),
        createTestingPinia({
          createSpy: vi.fn,
          initialState: {
            container: {
              containers: [
                new Container(
                  "123",
                  new Date("2026-01-03T00:00:00Z"),
                  new Date(),
                  new Date(),
                  "image",
                  "test",
                  "command",
                  "host",
                  {},
                  "running",
                  0,
                  0,
                  [],
                ),
                new Container(
                  "345",
                  new Date("2026-01-02T00:00:00Z"),
                  new Date(),
                  new Date(),
                  "image",
                  "foo bar",
                  "command",
                  "host",
                  {},
                  "running",
                  0,
                  0,
                  [],
                ),
                new Container(
                  "567",
                  new Date("2026-01-01T00:00:00Z"),
                  new Date(),
                  new Date(),
                  "image",
                  "baz",
                  "command",
                  "host",
                  {},
                  "running",
                  0,
                  0,
                  [],
                ),
                ...extra,
              ],
            },
          },
        }),
      ],
    },
  });
  return wrapper;
}

/**
 * @vitest-environment jsdom
 */
describe("<FuzzySearchModal />", () => {
  vi.mocked(useRouter).mockReturnValue({
    ...router,
    push: vi.fn(),
  });

  // The page under the palette decides what merge can do; tests set it per case.
  const route = { name: "/" as string, params: {} as Record<string, string> };
  vi.mocked(useRoute).mockReturnValue(route as unknown as ReturnType<typeof useRoute>);

  beforeEach(() => {
    vi.mocked(useRouter().push).mockReset();
    route.name = "/";
    route.params = {};
  });

  test("suggests recent containers before typing", async () => {
    const wrapper = createFuzzySearchModal();
    expect(wrapper.findAll("ul [data-name]").map((el) => el.text())).toEqual(["test", "foo bar", "baz"]);
  });

  test("search for foo", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.findAll("ul [data-name]").length).toBe(1);
    expect(wrapper.find("ul [data-name]").html()).toMatchInlineSnapshot(
      `"<span data-v-2818ba83="" class="text-base-content" data-name=""><mark>foo</mark> bar</span>"`,
    );
  });

  test("escapes html in names from container labels", async () => {
    const payload = 'pwn<iframe srcdoc="x"></iframe>';
    const wrapper = createFuzzySearchModal([
      new Container(
        "789",
        new Date("2026-01-04T00:00:00Z"),
        new Date(),
        new Date(),
        "image",
        payload,
        "command",
        "host",
        {},
        "running",
        0,
        0,
        [],
      ),
    ]);
    expect(wrapper.find("ul iframe").exists()).toBe(false);

    await wrapper.find("input").setValue("pwn");
    const name = wrapper.find("ul [data-name]");
    expect(name.find("iframe").exists()).toBe(false);
    expect(name.find("mark").text()).toBe("pwn");
    expect(name.text()).toBe(payload);
  });

  test("tells the user when nothing matches", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("nothing-matches-this");
    expect(wrapper.findAll("ul [data-name]").length).toBe(0);
    expect(wrapper.find("[data-testid=no-matches]").exists()).toBe(true);
  });

  test("log search row sends unlinked instances to cloud settings", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("nothing-matches-this");
    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith("/settings/cloud");
  });

  test("typing does not run a cloud search behind the palette", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}"));
    vi.stubGlobal("fetch", fetchMock);
    useCloudConfig().cloudConfig.value = { linked: true, streamLogs: true } as never;
    try {
      const wrapper = createFuzzySearchModal();
      await wrapper.find("input").setValue("timeout");
      await new Promise((r) => setTimeout(r, 400));
      const searched = fetchMock.mock.calls.some(([url]) => String(url).includes("/api/cloud/search"));
      expect(searched).toBe(false);
      // The row still offers the search, and Shift+Enter still runs it.
      await wrapper.find("input").trigger("keydown", { key: "Enter", shiftKey: true });
      expect(useRouter().push).toHaveBeenCalledWith({ path: "/cloud/search", query: { q: "timeout" } });
    } finally {
      useCloudConfig().cloudConfig.value = null;
      vi.unstubAllGlobals();
    }
  });

  test("choose baz", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("baz");
    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/container/[id]", params: { id: "567" } });
    expect(wrapper.emitted("close")).toHaveLength(1);
  });

  test("offers no merge when nothing is streaming", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.find("button[aria-pressed]").exists()).toBe(false);
    await wrapper.find("input").trigger("keydown", { key: "Enter", metaKey: true });
    expect(useRouter().push).not.toHaveBeenCalled();
  });

  test("cmd+enter merges the selected container into the stream on screen", async () => {
    route.name = "/container/[id]";
    route.params = { id: "123" };
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.find("button[aria-pressed=false]").exists()).toBe(true);
    await wrapper.find("input").trigger("keydown", { key: "Enter", metaKey: true });
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/merged/[ids]", params: { ids: "123,345" } });
  });

  test("merges a container from another host", async () => {
    route.name = "/container/[id]";
    route.params = { id: "123" };
    const wrapper = createFuzzySearchModal([
      new Container(
        "999",
        new Date("2026-01-04T00:00:00Z"),
        new Date(),
        new Date(),
        "image",
        "remote",
        "command",
        "other-host",
        {},
        "running",
        0,
        0,
        [],
      ),
    ]);
    await wrapper.find("input").setValue("remote");
    expect(wrapper.find("button[aria-pressed=false]").exists()).toBe(true);
    await wrapper.find("input").trigger("keydown", { key: "Enter", metaKey: true });
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/merged/[ids]", params: { ids: "123,999" } });
  });

  test("cmd+enter on a merged container takes it back out", async () => {
    route.name = "/merged/[ids]";
    route.params = { ids: "123,345" };
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.find("button[aria-pressed=true]").exists()).toBe(true);
    await wrapper.find("input").trigger("keydown", { key: "Enter", ctrlKey: true });
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/container/[id]", params: { id: "123" } });
  });

  test("matches commands by keyword", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("theme");
    const items = wrapper.findAll("li").map((li) => li.text());
    expect(items).toContain("command-palette.theme-dark");
  });

  test("theme commands set the theme explicitly", async () => {
    lightTheme.value = "auto";
    const wrapper = createFuzzySearchModal();

    await wrapper.find("input").setValue("dark theme");
    await wrapper.find("input").trigger("keydown.enter");
    expect(lightTheme.value).toBe("dark");

    await wrapper.find("input").setValue("light theme");
    await wrapper.find("input").trigger("keydown.enter");
    expect(lightTheme.value).toBe("light");

    await wrapper.find("input").setValue("system theme");
    await wrapper.find("input").trigger("keydown.enter");
    expect(lightTheme.value).toBe("auto");
  });
});
