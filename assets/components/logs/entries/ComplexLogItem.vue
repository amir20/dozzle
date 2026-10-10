<template>
  <DefineTemplate v-slot="{ data }">
    <ul class="inline-flex flex-wrap space-x-4" @click="preventDefaultOnLinks">
      <li v-for="(value, name) in data" :key="name" v-if="isObject(data)">
        <span class="key">{{ name }}=</span>
        <span class="value" v-if="value === null">&lt;null&gt;</span>
        <ReuseTemplate :data="value" v-else-if="isObject(value) || Array.isArray(value)" />
        <span v-else class="value" :class="typeof value" v-html="stripAnsi(String(value))"></span>
      </li>
      <li v-else-if="Array.isArray(data)">
        <ul class="array inline-flex flex-wrap space-x-1">
          <li
            v-for="(item, index) in data"
            :key="index"
            class="after:text-base-content/70 not-last:after:content-[',']"
          >
            <ReuseTemplate :data="item" v-if="isObject(item) || Array.isArray(item)" />
            <span v-else class="value" :class="typeof item" v-html="stripAnsi(String(item))"></span>
          </li>
        </ul>
      </li>
      <li class="key" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
    </ul>
  </DefineTemplate>
  <LogItem :logEntry>
    <LogLevel class="flex select-none" :level="logEntry.level" :event="logEntry.matchedEvent" />
    <!-- min-w-0 lets a long unbroken value (a token, an id) wrap instead of
         widening the row and pushing the memory chip off the right edge. -->
    <div @click="containers.length > 0 && showDrawer(LogDetails, { entry: logEntry })" class="min-w-0 cursor-pointer">
      <ReuseTemplate :data="validValues" />
    </div>
    <PatternMemoryChip v-if="logEntry.patternMemory" :memory="logEntry.patternMemory" :log-entry="logEntry" />
  </LogItem>
</template>
<script lang="ts" setup>
import stripAnsi from "strip-ansi";
import { type ComplexLogEntry } from "@/models/LogEntry";
const LogDetails = defineAsyncComponent(() => import("./LogDetails.vue"));

const { logEntry } = defineProps<{
  logEntry: ComplexLogEntry;
  showContainerName?: boolean;
}>();

const { containers } = useLoggingContext();

const [DefineTemplate, ReuseTemplate] = createReusableTemplate();

const validValues = computed(() => {
  return Object.fromEntries(Object.entries(logEntry.message).filter(([_, value]) => value !== undefined));
});

const showDrawer = useDrawer();
function preventDefaultOnLinks(event: MouseEvent) {
  if (event.target instanceof HTMLAnchorElement && event.target.rel?.includes("external")) {
    event.stopImmediatePropagation();
  }
}
</script>

<style scoped>
@reference "@/main.css";
.key {
  @apply text-base-content/70 font-light;
}

.value {
  @apply text-base-content font-bold [overflow-wrap:anywhere];
}

.array {
  @apply before:text-base-content/80 after:text-base-content/80 before:content-['['] after:content-[']'];
}

.string {
  @apply before:content-['"'] after:content-['"'];
}
</style>
