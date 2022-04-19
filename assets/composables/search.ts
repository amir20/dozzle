import { ref, computed, Ref } from "vue";

const searchFilter = ref<string>("");
const showSearch = ref(false);

import type { LogEntry } from "@/types/LogEntry";

export function useSearchFilter() {
  const regex = computed(() => {
    const isSmartCase = searchFilter.value === searchFilter.value.toLowerCase();
    return isSmartCase ? new RegExp(searchFilter.value, "i") : new RegExp(searchFilter.value);
  });

  function filteredMessages(messages: Ref<LogEntry[]>) {
    return computed(() => {
      if (searchFilter && searchFilter.value) {
        try {
          return messages.value.filter((d) => d.message.match(regex.value));
        } catch (e) {
          if (e instanceof SyntaxError) {
            console.info(`Ignoring SyntaxError from search.`, e);
            return messages.value;
          }
          throw e;
        }
      }

      return messages.value;
    });
  }

  function markSearch(log: string) {
    if (searchFilter && searchFilter.value) {
      return log.replace(regex.value, `<mark>$&</mark>`);
    }
    return log;
  }

  function resetSearch() {
    searchFilter.value = "";
    showSearch.value = false;
  }

  function isSearching() {
    return showSearch.value && searchFilter.value;
  }

  return {
    filteredMessages,
    searchFilter,
    showSearch,
    markSearch,
    resetSearch,
    isSearching,
  };
}
