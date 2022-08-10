import { ref, computed, Ref } from "vue";

const searchFilter = ref<string>("");
const showSearch = ref(false);

import { VisibleLogEntry } from "@/types/VisibleLogEntry";

function matchPayload(payload: Record<string, any>, regex: RegExp) {
  for (const key in payload) {
    const value = payload[key];
    if (typeof value === "string" && regex.test(value)) {
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
      if (searchFilter && searchFilter.value) {
        try {
          return messages.value.filter((d) => {
            if (d.entry.payload) {
              return matchPayload(d.entry.payload, regex.value);
            } else if (d.entry.message) {
              return regex.value.test(d.entry.message);
            }
            throw new Error("No message or payload");
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
