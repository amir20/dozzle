<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogStd :std="logEntry.std" class="select-none" v-if="showStd" />
    <ContainerName class="shrink-0 select-none" :id="logEntry.containerID" v-if="showContainerName" />
    <LogDate :date="logEntry.date" v-if="showTimestamp" class="select-none" />
    <LogLevel
      class="flex select-none"
      :level="logEntry.level"
      :position="logEntry instanceof SimpleLogEntry ? logEntry.position : undefined"
    />
    <slot />
  </div>
</template>
<script lang="ts" setup>
import { LogEntry, SimpleLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: LogEntry<any>;
}>();
const { showHostname, showContainerName } = useLoggingContext();
</script>
<style scoped lang="postcss">
.log-wrapper :deep(a) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
