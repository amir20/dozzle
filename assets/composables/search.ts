import { type Ref } from "vue";
import { type LogEntry, type JSONObject, SimpleLogEntry, ComplexLogEntry } from "@/models/LogEntry";
import { encodeXML } from "entities";

const searchFilter = ref<string>("");
const debouncedSearchFilter = useDebounce(searchFilter);
const showSearch = ref(false);

function matchRecord(record: Record<string, any>, regex: RegExp): boolean {
  for (const key in record) {
    const value = record[key];
    if (typeof value === "string" && regex.test(value)) {
      return true;
    }
    if (isObject(value) && regex.test(JSON.stringify(value))) {
      return true;
    }
    if (Array.isArray(value) && matchRecord(value, regex)) {
      return true;
    }
  }
  return false;
}

export function useSearchFilter() {
  const regex = $computed(() => {
    const isSmartCase = debouncedSearchFilter.value === debouncedSearchFilter.value.toLowerCase();
    return new RegExp(encodeXML(debouncedSearchFilter.value), isSmartCase ? "i" : undefined);
  });

  function filteredMessages(messages: Ref<LogEntry<string | JSONObject>[]>) {
    return computed(() => {
      if (debouncedSearchFilter.value && showSearch.value) {
        try {
          return messages.value.filter((d) => {
            if (d instanceof SimpleLogEntry) {
              return regex.test(d.message);
            } else if (d instanceof ComplexLogEntry) {
              return matchRecord(d.message, regex);
            }
            return false;
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

  function markSearch(log: { toString(): string }): string;
  function markSearch(log: string[]): string[];
  function markSearch(log: { toString(): string } | string[]) {
    if (!debouncedSearchFilter.value) {
      return log;
    }
    if (Array.isArray(log)) {
      return log.map((d) => markSearch(d));
    }

    return log.toString().replace(regex, (match) => `<mark>${match}</mark>`);
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
