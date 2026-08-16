<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogActions :logEntry :container />

    <LogStd :std="logEntry.std" class="shrink-0 select-none" v-if="showStd" />

    <!-- Cloud matched a notification on this exact line. Badged here rather
         than inserted as its own row: the event IS this line, so a separate
         marker would duplicate it — and during a storm would put one between
         every pair of lines. -->
    <span
      v-if="logEntry.matchedEvent"
      class="event-badge shrink-0 select-none"
      :data-suppressed="logEntry.matchedEvent.suppressed"
      :title="logEntry.matchedEvent.suppressed ? $t('label.event-suppressed-hint') : $t('label.event-sent-hint')"
    >
      <mdi:bell-off v-if="logEntry.matchedEvent.suppressed" class="size-3.5" />
      <mdi:bell-alert v-else class="size-3.5" />
    </span>

    <div class="flex gap-x-2 gap-y-1 group-[.compact]:gap-y-0 has-[>_*:nth-of-type(2)]:flex-col-reverse md:flex-row!">
      <RandomColorTag class="w-30 shrink-0 select-none md:w-40" :value="host.name" v-if="showHostname" />
      <RandomColorTag
        v-if="showContainerName"
        class="w-30 shrink-0 select-none group-[.compact]:flex-1 md:w-40"
        :value="container.id"
        truncateRight
      >
        {{ container.name }}
      </RandomColorTag>
      <LogDate v-if="showTimestamp" :date="logEntry.date" class="shrink-0 select-none" />
    </div>
    <slot />
  </div>
</template>
<script lang="ts" setup>
import { LogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: LogEntry<any>;
}>();
const { showHostname, showContainerName } = useLoggingContext();

const { currentContainer } = useContainerStore();
const { hosts } = useHosts();

const container = currentContainer(toRef(() => logEntry.containerID));
const host = computed(() => hosts.value[container.value.host]);
</script>

<style scoped>
@reference "@/main.css";
/* Quiet by default: a badge on every line of a storm must not out-shout the
   logs. The delivered one keeps the alert colour because it marks the moment
   something was actually sent. */
.event-badge {
  @apply mt-0.5 opacity-45 transition-opacity;
}
.event-badge:hover {
  @apply opacity-100;
}
.event-badge[data-suppressed="false"] {
  @apply text-error opacity-80;
}
</style>
