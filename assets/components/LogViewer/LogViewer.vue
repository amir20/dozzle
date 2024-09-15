<template>
  <SideDrawer ref="drawer">
    <Suspense>
      <component :is="component" v-bind="properties" />
      <template #fallback> Loading... </template>
    </Suspense>
  </SideDrawer>

  <LogList :messages="visibleMessages" :show-container-name="showContainerName" />
</template>

<script lang="ts" setup>
import SideDrawer from "@/components/common/SideDrawer.vue";
import { type JSONObject, LogEntry } from "@/models/LogEntry";
import LogAnalytics from "./LogAnalytics.vue";

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
const { component, properties, showDrawer } = createDrawer(drawer);

const visibleMessages = filteredPayload(messages);
const router = useRouter();
const { containers } = useLoggingContext();

onKeyStroke("f", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    if (containers.value.length == 1) {
      const container = containers.value[0];
      showDrawer(LogAnalytics, { container });
      e.preventDefault();
    }
  }
});

watchEffect(() => {
  const query = {} as Record<string, string>;
  if (debouncedSearchFilter.value !== "") {
    query.search = debouncedSearchFilter.value;
  }

  if (!streamConfig.value.stderr) {
    query.stderr = streamConfig.value.stderr.toString();
  }

  if (!streamConfig.value.stdout) {
    query.stdout = streamConfig.value.stdout.toString();
  }

  router.push({
    query,
    replace: true,
  });
});
</script>
<style scoped lang="postcss"></style>
