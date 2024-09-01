const searchQueryFilter = ref<string>("");
const debouncedSearchFilter = refDebounced(searchQueryFilter);
const showSearch = ref(false);

export function useSearchFilter() {
  function resetSearch() {
    searchQueryFilter.value = "";
    showSearch.value = false;
  }

  const isSearching = computed(() => showSearch.value && debouncedSearchFilter.value !== "");

  return {
    searchQueryFilter,
    debouncedSearchFilter,
    showSearch,
    resetSearch,
    isSearching,
  };
}
