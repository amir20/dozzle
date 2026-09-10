/**
 * @vitest-environment jsdom
 */
import { beforeEach, describe, expect, test } from "vitest";
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
  });
});
