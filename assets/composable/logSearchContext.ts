import { JSONObject, LogEntry } from "@/models/LogEntry";

const lastSelectedItem = ref<LogEntry<string | JSONObject> | undefined>(undefined);
export const useLogSearchContext = () => {
  const { resetSearch } = useSearchFilter();

  function handleJumpLineSelected(e: Event, item: LogEntry<string | JSONObject>) {
    console.log("handleJumpLineSelected", item);

    lastSelectedItem.value = item;
    console.log("lastSelectedItem", toRaw(lastSelectedItem));
    resetSearch();
  }

  return {
    lastSelectedItem,
    handleJumpLineSelected,
  };
};
