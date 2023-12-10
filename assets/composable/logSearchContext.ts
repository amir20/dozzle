import { JSONObject, LogEntry } from "@/models/LogEntry";

const lastSelectedItem = ref<LogEntry<string | JSONObject> | undefined>(undefined);
export const useLogSearchContext = () => {
  const { resetSearch } = useSearchFilter();

  function handleJumpLineSelected(e: Event, item: LogEntry<string | JSONObject>) {
    lastSelectedItem.value = item;
    resetSearch();
  }

  return {
    lastSelectedItem,
    handleJumpLineSelected,
  };
};
