<template>
  <LogList :messages="visibleMessages" />
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: Map<string[], boolean>;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const visibleMessages = filteredPayload(messages);
</script>
<style scoped></style>
