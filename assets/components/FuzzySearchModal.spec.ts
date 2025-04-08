import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";

import FuzzySearchModal from "./FuzzySearchModal.vue";

import { Container } from "@/models/Container";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { router } from "@/modules/router";

// @ts-ignore
import EventSource, { sources } from "eventsourcemock";

vi.mock("vue-router");

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
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
                  new Date(),
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
                  new Date(),
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

  test("shows none initially", async () => {
    const wrapper = createFuzzySearchModal();
    expect(wrapper.findAll("li").length).toBe(0);
  });

  test("search for foo", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.findAll("li").length).toBe(1);
    expect(wrapper.find("ul [data-name]").html()).toMatchInlineSnapshot(
      `"<span data-v-dc2e8c61="" data-name=""><mark>foo</mark> bar</span>"`,
    );
  });

  test("choose baz", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("baz");
    await wrapper.find("input").trigger("keydown.enter");
    expect(useRouter().push).toHaveBeenCalledWith({ name: "/container/[id]", params: { id: "567" } });
  });
});
