<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogActions :logEntry :container />

    <LogStd :std="logEntry.std" class="shrink-0 select-none" v-if="showStd" />

    <!-- On a phone the tags stack under the timestamp, so they drop to the small
         size: a whole-size chip made every line two lines tall. -->
    <div
      class="flex gap-x-2 gap-y-0.5 group-[.compact]:gap-y-0 has-[>_*:nth-of-type(2)]:flex-col-reverse md:mr-1 md:flex-row! md:gap-y-1"
    >
      <RandomColorTag
        class="w-30 shrink-0 select-none max-md:text-xs max-md:leading-4 md:w-40"
        :value="host?.name ?? ''"
        v-if="showHostname"
      />
      <RandomColorTag
        v-if="showContainerName"
        class="w-30 shrink-0 select-none group-[.compact]:flex-1 max-md:text-xs max-md:leading-4 md:w-40"
        :value="logEntry.containerID"
        truncateRight
      >
        {{ container?.name ?? logEntry.containerID }}
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

// A line can outlive its container: an update or recreate replaces it under a new
// id while the lines it already wrote stay on screen.
const container = currentContainer(toRef(() => logEntry.containerID));
const host = computed(() => (container.value ? hosts.value[container.value.host] : undefined));
</script>
