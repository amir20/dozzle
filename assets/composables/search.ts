import { ref, computed, Ref } from "vue";

const searchFilter = ref<string>();
const showSearch = ref(false);

import type { LogEntry } from "@/types/LogEntry";

export function useSearchFilter() {
  function filteredMessages(messages: Ref<LogEntry[]>) {
    return computed(() => {
      if (searchFilter && searchFilter.value) {
        const isSmartCase = searchFilter.value === searchFilter.value.toLowerCase();
        try {
          const regex = isSmartCase ? new RegExp(searchFilter.value, "i") : new RegExp(searchFilter.value);
          return messages.value
            .filter((d) => d.message.match(regex))
            .map((d) => ({
              ...d,
              message: d.message.replace(regex, "<mark>$&</mark>"),
            }));
        } catch (e) {
          if (e instanceof SyntaxError) {
            console.info(`Ignoring SytaxError from search.`, e);
            return messages.value;
          }
          throw e;
        }
      }

      return messages.value;
    });
  }

  return {
    filteredMessages,
    searchFilter,
    showSearch
  };
}
