<template>
  <SideDrawer ref="drawer">
    <LogDetails :entry="entry" v-if="entry && entry instanceof ComplexLogEntry" />
  </SideDrawer>
  <LogList
    :messages="visibleMessages"
    :last-selected-item="lastSelectedItem"
    :show-container-name="showContainerName"
  />
</template>

<script lang="ts" setup>
import { useRouteHash } from "@vueuse/router";
import SideDrawer from "@/components/common/SideDrawer.vue";
import { ComplexLogEntry, type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: Map<string[], boolean>;
  showContainerName: boolean;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { debouncedSearchFilter } = useSearchFilter();
const { streamConfig } = useLoggingContext();

const drawer = ref<InstanceType<typeof SideDrawer>>() as Ref<InstanceType<typeof SideDrawer>>;

const { entry } = provideLogDetails(drawer);

const visibleMessages = filteredPayload(messages);

const { lastSelectedItem } = useLogSearchContext() as {
  lastSelectedItem: Ref<LogEntry<string | JSONObject> | undefined>;
};
const routeHash = useRouteHash();
watch(
  routeHash,
  (hash) => {
    if (hash) {
      document.querySelector(`[data-key="${hash.substring(1)}"]`)?.scrollIntoView({ block: "center" });
    }
  },
  { immediate: true, flush: "post" },
);

const router = useRouter();

watchArray(
  [debouncedSearchFilter, streamConfig],
  () => {
    router.push({
      query: {
        search: debouncedSearchFilter.value,
        stderr: streamConfig.value.stderr.toString(),
        stdout: streamConfig.value.stdout.toString(),
      },
      replace: true,
    });
  },
  { deep: true },
);
</script>
<style scoped lang="postcss"></style>
