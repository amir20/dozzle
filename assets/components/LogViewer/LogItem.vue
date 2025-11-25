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
        :class="{ 'bg-secondary': isTargetLine }"
      />
    </div>
    <LogLevel
      class="flex select-none"
      :level="logEntry.level"
      :position="logEntry instanceof SimpleLogEntry ? logEntry.position : undefined"
    />
    <slot />

    <!-- Context navigation controls for target line -->
    <div v-if="isTargetLine" class="ml-auto flex shrink-0 items-center gap-1">
      <button
        @click="scrollByLines(-10)"
        class="btn btn-xs btn-ghost border-base-content/20 border"
        title="Scroll 10 lines up"
      >
        <mdi:chevron-up />10
      </button>
      <button
        @click="scrollByLines(10)"
        class="btn btn-xs btn-ghost border-base-content/20 border"
        title="Scroll 10 lines down"
      >
        <mdi:chevron-down />10
      </button>
      <button
        @click="scrollByLines(-50)"
        class="btn btn-xs btn-ghost border-base-content/20 border"
        title="Scroll 50 lines up"
      >
        <mdi:chevron-double-up />50
      </button>
      <button
        @click="scrollByLines(50)"
        class="btn btn-xs btn-ghost border-base-content/20 border"
        title="Scroll 50 lines down"
      >
        <mdi:chevron-double-down />50
      </button>
    </div>
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

const isTargetLine = computed(() => route.query.logId === logEntry.id.toString());

function scrollByLines(offset: number) {
  const logList = document.querySelector("[data-logs]");
  if (!logList) return;

  const items = Array.from(logList.querySelectorAll("li[id]"));
  const currentIndex = items.findIndex((el) => el.id === logEntry.id.toString());

  if (currentIndex === -1) return;

  const targetIndex = Math.max(0, Math.min(items.length - 1, currentIndex + offset));
  const targetElement = items[targetIndex];

  targetElement?.scrollIntoView({ behavior: "smooth", block: "center" });
}
</script>
