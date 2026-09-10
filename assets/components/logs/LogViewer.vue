<template>
  <LogList :messages="visibleMessages" />
</template>

<script lang="ts" setup>
import { type LogMessage, LogEntry } from "@/models/LogEntry";
import type { VisibleKeysSource } from "@/composable/logs/visible";

const props = defineProps<{
  messages: LogEntry<LogMessage>[];
  visibleKeys: VisibleKeysSource;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const visibleMessages = filteredPayload(messages);

// What is on screen is what a chat turn carries, so the assistant reads the
// same window the person is looking at.
if (isViewContextOwner()) publishVisibleLogs(visibleMessages);
</script>
<style scoped></style>
