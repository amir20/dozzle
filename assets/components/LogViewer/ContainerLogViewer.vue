<template>
  <log-viewer :messages="filtered" :last-selected-item="lastSelectedItem" :visible-keys="visibleKeys" />
</template>

<script lang="ts" setup>
import { useRouteHash } from "@vueuse/router";

import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
}>();

const { container } = useContainerContext();

const visibleKeys = persistentVisibleKeys(container);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { filteredMessages } = useSearchFilter();

const { messages } = toRefs(props);
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
