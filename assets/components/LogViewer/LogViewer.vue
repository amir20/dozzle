<template>
  <LogList :messages="visibleMessages" />
</template>

<script lang="ts" setup>
import { type LogMessage, LogEntry } from "@/models/LogEntry";
import type { VisibleKeysSource } from "@/composable/visible";

const props = defineProps<{
  messages: LogEntry<LogMessage>[];
  visibleKeys: VisibleKeysSource;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const visibleMessages = filteredPayload(messages);
</script>
<style scoped></style>
