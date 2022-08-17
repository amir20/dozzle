import { ref, computed, Ref } from "vue";

const searchFilter = ref<string>("");
const showSearch = ref(false);

import { VisibleLogEntry } from "@/types/VisibleLogEntry";

function matchRecord(record: Record<string, any>, regex: RegExp): boolean {
  for (const key in record) {
    const value = record[key];
    if (typeof value === "string" && regex.test(value)) {
      return true;
    }
    if (Array.isArray(value) && matchRecord(value, regex)) {
      return true;
    }
  }
  return false;
}

export function useSearchFilter() {
  const regex = computed(() => {
    const isSmartCase = searchFilter.value === searchFilter.value.toLowerCase();
    return isSmartCase ? new RegExp(searchFilter.value, "i") : new RegExp(searchFilter.value);
  });

  function filteredMessages(messages: Ref<VisibleLogEntry[]>) {
    return computed(() => {
      if (searchFilter.value) {
        try {
          return messages.value.filter((d) => {
            if (d.isSimple()) {
              return regex.value.test(d.message);
            } else if (d.isComplex()) {
              return matchRecord(d.message, regex.value);
            }
            throw new Error("Unknown message type");
          });
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

  function markSearch(log: string): string;
  function markSearch(log: string[]): string[];
  function markSearch(log: string | string[]) {
    if (!searchFilter.value) {
      return log;
    }
    if (Array.isArray(log)) {
      return log.map((d) => markSearch(d));
    }

    return log.toString().replace(regex.value, (match) => `<mark>${match}</mark>`);
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
