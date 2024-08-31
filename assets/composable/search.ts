import { encodeXML } from "entities";

const searchFilter = ref<string>("");
const debouncedSearchFilter = useDebounce(searchFilter);
const showSearch = ref(false);

export function useSearchFilter() {
  const regex = computed(() => {
    const isSmartCase = debouncedSearchFilter.value === debouncedSearchFilter.value.toLowerCase();
    return new RegExp(encodeXML(debouncedSearchFilter.value), isSmartCase ? "i" : "");
  });

  function markSearch(log: { toString(): string }): string;
  function markSearch(log: string[]): string[];
  function markSearch(log: { toString(): string } | string[]) {
    if (!debouncedSearchFilter.value) {
      return log;
    }
    if (Array.isArray(log)) {
      return log.map((d) => markSearch(d));
    }

    const globalRegex = new RegExp(regex.value.source, regex.value.flags + "g");
    return log.toString().replaceAll(globalRegex, (match) => `<mark>${match}</mark>`);
  }

  function resetSearch() {
    searchFilter.value = "";
    showSearch.value = false;
  }

  const isSearching = computed(() => showSearch.value && searchFilter.value !== "");

  return {
    searchFilter,
    showSearch,
    markSearch,
    resetSearch,
    isSearching,
  };
}
