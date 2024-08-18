<template>
  <SideDrawer ref="drawer">
    <LogDetails :entry="entry" v-if="entry" />
  </SideDrawer>
  <LogList
    :messages="filtered"
    :last-selected-item="lastSelectedItem"
    :visible-keys="visibleKeys"
    :show-container-name="showContainerName"
  />
</template>

<script lang="ts" setup>
import { useRouteHash } from "@vueuse/router";
import SideDrawer from "@/components/common/SideDrawer.vue";
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: string[][];
  showContainerName: boolean;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages } = useSearchFilter();

const drawer = ref<InstanceType<typeof SideDrawer>>() as Ref<InstanceType<typeof SideDrawer>>;

const { entry } = provideLogDetails(drawer);

const visible = filteredPayload(messages);
const filtered = filteredMessages(visible);

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
</script>
<style scoped lang="postcss"></style>
