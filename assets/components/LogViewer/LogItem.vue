<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogActions :logEntry :container />

    <LogStd :std="logEntry.std" class="shrink-0 select-none" v-if="showStd" />

    <div class="flex gap-x-2 gap-y-1 group-[.compact]:gap-y-0 has-[>_*:nth-of-type(2)]:flex-col-reverse md:flex-row!">
      <RandomColorTag class="w-30 shrink-0 select-none md:w-40" :value="host.name" v-if="showHostname" />
      <RandomColorTag
        v-if="showContainerName"
        class="w-30 shrink-0 select-none group-[.compact]:flex-1 md:w-40"
        :value="container.name"
        truncateRight
      />
      <LogDate
        v-if="showTimestamp"
        :date="logEntry.date"
        class="shrink-0 select-none"
        :class="{ 'bg-secondary': route.query.logId === logEntry.id.toString() }"
      />
    </div>
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

const route = useRoute();
</script>
