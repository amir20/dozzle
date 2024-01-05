import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";

import FuzzySearchModal from "./FuzzySearchModal.vue";

import { Container } from "@/models/Container";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";

// @ts-ignore
import EventSource, { sources } from "eventsourcemock";

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
                new Container("123", new Date(), "image", "test", "command", "host", {}, "status", "running"),
                new Container("123", new Date(), "image", "foo bar", "command", "host", {}, "status", "running"),
                new Container("123", new Date(), "image", "baz", "command", "host", {}, "status", "exited"),
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
  test("shows all", async () => {
    const wrapper = createFuzzySearchModal();
    expect(wrapper.findAll("li").length).toBe(3);
  });

  test("search for foo", async () => {
    const wrapper = createFuzzySearchModal();
    await wrapper.find("input").setValue("foo");
    expect(wrapper.findAll("li").length).toBe(1);
    expect(wrapper.find("ul [data-name]").html()).toMatchInlineSnapshot(
      `"<span data-v-dc2e8c61="" data-name=""><mark>foo</mark> bar</span>"`,
    );
  });
});
