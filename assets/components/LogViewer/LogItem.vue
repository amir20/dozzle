<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogStd :std="logEntry.std" class="shrink-0 select-none" v-if="showStd" />
    <RandomColorTag class="shrink-0 select-none" :value="host.name" v-if="showHostname" />
    <RandomColorTag class="shrink-0 select-none" :value="container.name" v-if="showContainerName" truncateRight />
    <LogDate :date="logEntry.date" v-if="showTimestamp" class="shrink-0 select-none" />
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

const { currentContainer } = useContainerStore();
const { hosts } = useHosts();

const container = currentContainer(toRef(() => logEntry.containerID));
const host = computed(() => hosts.value[container.value.host]);
</script>
