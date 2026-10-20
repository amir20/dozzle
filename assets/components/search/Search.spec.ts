/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import Search from "./Search.vue";
import { useSearchFilter } from "@/composable/logs/search";

const cloud = vi.hoisted(() => ({ available: { value: false }, open: vi.fn() }));
vi.mock("@/composable/cloud/cloudLogSearch", async () => {
  const { ref } = await import("vue");
  const available = ref(false);
  cloud.available = available;
  return { useCloudSearchHandoff: () => ({ available, open: cloud.open }) };
});

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
    cloud.available.value = false;
    cloud.open.mockReset();
  });

  test("flags an invalid regex on the box without recoloring it", async () => {
    const wrapper = mountSearch();
    search.searchQueryFilter.value = "valid";
    await nextTick();
    expect(wrapper.find('[data-testid="search-box"]').attributes("data-invalid")).toBeUndefined();

    search.searchQueryFilter.value = "[";
    await nextTick();
    expect(wrapper.find('[data-testid="search-box"]').attributes("data-invalid")).toBe("true");
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

  test("hides the cloud button when cloud search is not available", async () => {
    const wrapper = mountSearch();
    search.searchQueryFilter.value = "error";
    await nextTick();
    expect(wrapper.find('[data-testid="search-cloud"]').exists()).toBe(false);
  });

  test("hands the query to cloud from the button and from shift+enter", async () => {
    cloud.available.value = true;
    const wrapper = mountSearch();
    expect(wrapper.find('[data-testid="search-cloud"]').exists()).toBe(false);

    search.searchQueryFilter.value = "error";
    await nextTick();
    await wrapper.find('[data-testid="search-cloud"]').trigger("click");
    expect(cloud.open).toHaveBeenCalledWith("error");

    await wrapper.find("input").trigger("keydown", { key: "Enter", shiftKey: true });
    expect(cloud.open).toHaveBeenCalledTimes(2);
  });

  test("hides the cloud button for an inverse or broken pattern", async () => {
    cloud.available.value = true;
    const wrapper = mountSearch();
    search.searchQueryFilter.value = "[";
    await nextTick();
    expect(wrapper.find('[data-testid="search-cloud"]').exists()).toBe(false);

    search.searchQueryFilter.value = "error";
    search.inverseFilter.value = true;
    await nextTick();
    expect(wrapper.find('[data-testid="search-cloud"]').exists()).toBe(false);
    await wrapper.find("input").trigger("keydown", { key: "Enter", shiftKey: true });
    expect(cloud.open).not.toHaveBeenCalled();
  });
});
