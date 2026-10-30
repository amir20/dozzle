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
import { useImageUpdates } from "@/composable/containers/imageUpdate";
import config from "@/stores/config";
import { compact } from "@/stores/settings";

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
    localStorage.clear();
    // jsdom has no layout, so nothing scrolls.
    Element.prototype.scrollIntoView = vi.fn();
  });

  // A row's title, without the pills and shortcut keys after it. Container names
  // are not in a span, so this lists commands only.
  const texts = (wrapper: ReturnType<typeof createFuzzySearchModal>) =>
    wrapper.findAll("li span.truncate").map((el) => el.text());

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

  test("stop asks for enter a second time", async () => {
    route.name = "/container/[id]";
    route.params = { id: "123" };
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}"));
    vi.stubGlobal("fetch", fetchMock);
    try {
      const wrapper = createFuzzySearchModal();
      await wrapper.find("input").setValue(">stop");
      await wrapper.find("input").trigger("keydown.enter");
      expect(wrapper.find("[data-testid=armed]").text()).toBe("command-palette.confirm-stop");
      expect(wrapper.emitted("close")).toBeUndefined();

      await wrapper.find("input").trigger("keydown.enter");
      expect(wrapper.emitted("close")).toHaveLength(1);
      expect(fetchMock.mock.calls.some(([url]) => String(url).includes("/actions/stop"))).toBe(true);
    } finally {
      vi.unstubAllGlobals();
    }
  });

  test("a held enter key never confirms", async () => {
    route.name = "/container/[id]";
    route.params = { id: "123" };
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue(">stop");
    await wrapper.find("input").trigger("keydown", { key: "Enter" });
    await wrapper.find("input").trigger("keydown", { key: "Enter", repeat: true });
    expect(wrapper.find("[data-testid=armed]").exists()).toBe(true);
    expect(wrapper.emitted("close")).toBeUndefined();
  });

  test("a query reopened from cloud search is not read as a prefix", async () => {
    Object.assign(route, { path: "/cloud/search", query: { q: "@user" } });
    try {
      const wrapper = createFuzzySearchModal();
      expect(wrapper.find("[data-testid=no-matches]").exists()).toBe(true);
      expect(wrapper.text()).toContain("cloud-search.section-logs");
    } finally {
      Object.assign(route, { path: undefined, query: undefined });
    }
  });

  test("moving away disarms a confirm", async () => {
    route.name = "/container/[id]";
    route.params = { id: "123" };
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue(">stop");
    await wrapper.find("input").trigger("keydown.enter");
    expect(wrapper.find("[data-testid=armed]").exists()).toBe(true);
    await wrapper.find("input").trigger("keydown.down");
    expect(wrapper.find("[data-testid=armed]").exists()).toBe(false);
  });

  test("tab lists a container's actions and enter opens its logs", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("baz");
    await wrapper.find("input").trigger("keydown.tab");
    expect(wrapper.find("[data-testid=drill-crumb]").text()).toBe("baz");
    expect(texts(wrapper)[0]).toBe("command-palette.open-logs");
    expect(texts(wrapper)).toContain("command-palette.restart-container");

    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/container/[id]", params: { id: "567" } });
  });

  test("escape leaves the action list and restores the query", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("baz");
    await wrapper.find("input").trigger("keydown.tab");
    await wrapper.find("input").trigger("keydown.esc");
    expect(wrapper.find("[data-testid=drill-crumb]").exists()).toBe(false);
    expect((wrapper.find("input").element as HTMLInputElement).value).toBe("baz");
    expect(wrapper.emitted("close")).toBeUndefined();
  });

  test("> searches commands only", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue(">");
    expect(wrapper.findAll("ul [data-name]").length).toBe(0);
    expect(texts(wrapper)).toContain("command-palette.toggle-compact");
  });

  test("@ lists hosts and opens one", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("@");
    expect(wrapper.findAll("ul [data-name]").map((el) => el.text())).toEqual(["localhost"]);
    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/host/[id]", params: { id: "localhost" } });
  });

  test("toggles show their current value", async () => {
    compact.value = true;
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("compact");
    expect(wrapper.find("[data-testid=state]").text()).toBe("command-palette.state-on");
    compact.value = false;
    await wrapper.vm.$nextTick();
    expect(wrapper.find("[data-testid=state]").text()).toBe("command-palette.state-off");
  });

  test("a command used before is suggested before typing", async () => {
    const first = createFuzzySearchModal();
    expect(texts(first)).not.toContain("command-palette.toggle-timestamps");
    await first.find("input").setValue("timestamps");
    await first.find("input").trigger("keydown.enter");

    const second = createFuzzySearchModal();
    expect(texts(second)).toContain("command-palette.toggle-timestamps");
  });

  test("lists containers with an update waiting", async () => {
    Object.assign(config, { enableUpdates: true, imageCheckMode: "manual" });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            status: "update-available",
            image: "image",
            remoteDigest: "sha256:1",
            checkedAt: new Date().toISOString(),
          }),
        ),
      ),
    );
    try {
      const wrapper = createFuzzySearchModal();
      const baz = wrapper.findAll("ul [data-name]").length;
      expect(baz).toBe(3);
      const { check } = useImageUpdates();
      const container = new Container(
        "567",
        new Date(),
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
      );
      await check(container, true);
      await wrapper.vm.$nextTick();

      expect(wrapper.text()).toContain("command-palette.section-updates");
      expect(wrapper.find("[data-testid=update-dot]").exists()).toBe(true);
      expect(texts(wrapper)).toContain("command-palette.review-updates");
      // Listed once, under updates, not again under recent.
      expect(wrapper.findAll("ul [data-name]").filter((el) => el.text() === "baz")).toHaveLength(1);
    } finally {
      Object.assign(config, { enableUpdates: undefined, imageCheckMode: undefined });
      vi.unstubAllGlobals();
    }
  });
});
