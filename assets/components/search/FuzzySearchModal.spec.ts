import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";

import FuzzySearchModal from "./FuzzySearchModal.vue";

import { Container } from "@/models/Container";
import { lightTheme } from "@/stores/settings";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { router } from "@/modules/router";

// @ts-ignore
import EventSource, { sources } from "eventsourcemock";

vi.mock("vue-router");

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }], enableActions: true },
  withBase: (path: string) => path,
}));

function createFuzzySearchModal() {
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

  beforeEach(() => {
    vi.mocked(useRouter().push).mockReset();
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

  test("choose baz", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("baz");
    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/container/[id]", params: { id: "567" } });
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
