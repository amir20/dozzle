/**
 * @vitest-environment jsdom
 */
import { beforeEach, describe, expect, test, vi } from "vitest";
import { useSearchFilter } from "./search";

describe("useSearchFilter", () => {
  // State is a module-level singleton, so reset before each test.
  beforeEach(() => {
    useSearchFilter().resetSearch();
  });

  test("isValidQuery reflects regex validity", () => {
    const { searchQueryFilter, isValidQuery } = useSearchFilter();
    searchQueryFilter.value = "foo.*";
    expect(isValidQuery.value).toBe(true);

    searchQueryFilter.value = "[";
    expect(isValidQuery.value).toBe(false);
  });

  test("a half-typed pattern never reaches the stream", async () => {
    const { searchQueryFilter, showSearch, appliedSearchFilter, isSearching } = useSearchFilter();
    showSearch.value = true;

    searchQueryFilter.value = "error";
    await vi.waitFor(() => expect(appliedSearchFilter.value).toBe("error"));
    expect(isSearching.value).toBe(true);

    // Typing the opening half of a group leaves the matches on screen alone rather than
    // sending the server a pattern it answers with a 400.
    searchQueryFilter.value = "error|(warn";
    await vi.waitFor(() => expect(searchQueryFilter.value).toBe("error|(warn"));
    expect(appliedSearchFilter.value).toBe("error");
    expect(isSearching.value).toBe(true);

    searchQueryFilter.value = "error|(warn)";
    await vi.waitFor(() => expect(appliedSearchFilter.value).toBe("error|(warn)"));
  });

  test("clearing the query stops filtering", async () => {
    const { searchQueryFilter, showSearch, appliedSearchFilter, isSearching } = useSearchFilter();
    showSearch.value = true;
    searchQueryFilter.value = "error";
    await vi.waitFor(() => expect(isSearching.value).toBe(true));

    searchQueryFilter.value = "";
    await vi.waitFor(() => expect(appliedSearchFilter.value).toBe(""));
    expect(isSearching.value).toBe(false);
  });

  test("closing the box stops filtering even with a query still in it", async () => {
    const { searchQueryFilter, showSearch, appliedSearchFilter, isSearching } = useSearchFilter();
    showSearch.value = true;
    searchQueryFilter.value = "error";
    await vi.waitFor(() => expect(isSearching.value).toBe(true));

    showSearch.value = false;
    await vi.waitFor(() => expect(appliedSearchFilter.value).toBe(""));
    expect(isSearching.value).toBe(false);
  });

  test("toggleInverse flips the inverse flag", () => {
    const { inverseFilter, toggleInverse } = useSearchFilter();
    expect(inverseFilter.value).toBe(false);
    toggleInverse();
    expect(inverseFilter.value).toBe(true);
    toggleInverse();
    expect(inverseFilter.value).toBe(false);
  });

  test("resetSearch clears query, visibility and inverse", () => {
    const { searchQueryFilter, showSearch, inverseFilter, toggleInverse, resetSearch } = useSearchFilter();
    searchQueryFilter.value = "abc";
    showSearch.value = true;
    toggleInverse();

    resetSearch();

    expect(searchQueryFilter.value).toBe("");
    expect(showSearch.value).toBe(false);
    expect(inverseFilter.value).toBe(false);
    expect(useSearchFilter().appliedSearchFilter.value).toBe("");
  });
});
