const searchQueryFilter = ref<string>("");
const debouncedSearchFilter = refDebounced(searchQueryFilter);
const showSearch = ref(false);

const searchParams = new URLSearchParams(window.location.search);
if (searchParams.get("search") !== null && searchParams.get("search") !== "") {
  searchQueryFilter.value = searchParams.get("search") || "";
  showSearch.value = true;
}
function resetSearch() {
  searchQueryFilter.value = "";
  showSearch.value = false;
}

const isSearching = computed(() => showSearch.value && debouncedSearchFilter.value !== "");

const isValidQuery = computed(() => {
  try {
    new RegExp(searchQueryFilter.value);
    return true;
  } catch (e) {
    return false;
  }
});

export function useSearchFilter() {
  return {
    searchQueryFilter,
    isValidQuery,
    debouncedSearchFilter,
    showSearch,
    resetSearch,
    isSearching,
  };
}
