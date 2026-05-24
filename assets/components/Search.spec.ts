/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test } from "vitest";
import { createI18n } from "vue-i18n";
import Search from "./Search.vue";
import { useSearchFilter } from "@/composable/search";

const search = useSearchFilter();

function mountSearch() {
  search.showSearch.value = true;
  return mount(Search, {
    global: { plugins: [createI18n({})] },
  });
}

describe("<Search />", () => {
  beforeEach(() => {
    search.resetSearch();
  });

  test("flags an invalid regex with a warning style", async () => {
    const wrapper = mountSearch();
    search.searchQueryFilter.value = "valid";
    await nextTick();
    expect(wrapper.find(".input").classes()).not.toContain("input-warning");

    search.searchQueryFilter.value = "[";
    await nextTick();
    expect(wrapper.find(".input").classes()).toContain("input-warning");
  });

  test("binds the input to the shared search filter", async () => {
    const wrapper = mountSearch();
    await wrapper.find("input").setValue("abc");
    expect(search.searchQueryFilter.value).toBe("abc");
  });

  test("toggles the inverse filter", async () => {
    const wrapper = mountSearch();
    expect(search.inverseFilter.value).toBe(false);
    await wrapper.find("button").trigger("click");
    expect(search.inverseFilter.value).toBe(true);
  });

  test("escape resets the search", async () => {
    const wrapper = mountSearch();
    search.searchQueryFilter.value = "abc";
    await wrapper.find("input").trigger("keyup.esc");
    expect(search.searchQueryFilter.value).toBe("");
    expect(search.showSearch.value).toBe(false);
  });
});
