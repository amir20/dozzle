import { JSONObject, LogEntry } from "@/models/LogEntry";
import SideDrawer from "@/components/common/SideDrawer.vue";

export const showLogDetails = Symbol("showLogDetails") as InjectionKey<
  (logEntry: LogEntry<string | JSONObject>) => void
>;

export const provideLogDetails = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const entry = ref<LogEntry<string | JSONObject>>();

  provide(showLogDetails, (logEntry: LogEntry<string | JSONObject>) => {
    entry.value = logEntry;
    drawer.value?.open();
  });

  return { entry };
};

export const useLogDetails = () => {
  const showDetails = inject(showLogDetails);
  if (!showDetails) {
    throw new Error("No showLogDetails provided");
  }
  return showDetails;
};
