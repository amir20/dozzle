<template>
  <DefineTemplate v-slot="{ data, top }">
    <ul class="inline-flex flex-wrap space-x-4" @click="preventDefaultOnLinks">
      <!-- Only the line's own time and level repeat the row; a nested one is just data. -->
      <li
        v-for="(value, name) in data"
        :key="name"
        v-if="isObject(data)"
        :class="{ repeated: top && isRepeated(name) }"
      >
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
      <!-- Readable: what the line says first, then the rest in its own order. -->
      <div v-if="reading" class="headline">
        <template v-if="reading.method !== undefined || reading.status !== undefined">
          <span v-if="reading.method" class="font-bold" v-html="reading.method"></span>
          <span v-if="reading.path" v-html="stripAnsi(reading.path)"></span>
          <span v-if="reading.status !== undefined" class="font-bold" :class="statusClass(reading.status)">{{
            reading.status
          }}</span>
          <span v-if="reading.durationMs !== undefined" :class="{ 'text-warning': reading.durationMs > 1000 }">{{
            formatDuration(reading.durationMs)
          }}</span>
        </template>
        <!-- v-html like every other value here: the server escapes them, and a
             search match arrives wrapped in <mark>. -->
        <span v-if="reading.title" class="font-bold" v-html="stripAnsi(reading.title)"></span>
        <span v-if="reading.error" class="text-error" v-html="stripAnsi(reading.error)"></span>
      </div>
      <ReuseTemplate :data="shownValues" top v-if="Object.keys(shownValues).length > 0 || !reading" />
    </div>
    <PatternMemoryChip v-if="logEntry.patternMemory" :memory="logEntry.patternMemory" :log-entry="logEntry" />
  </LogItem>
</template>
<script lang="ts" setup>
import stripAnsi from "strip-ansi";
import { type ComplexLogEntry } from "@/models/LogEntry";
import { formatDuration, LEVEL_KEYS, readFields, TIME_KEYS } from "@/utils/readable";
import { lineStyle, showTimestamp } from "@/stores/settings";
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

// Readable only, and only when there is something to lead with: a line with no
// message, error or request reads best as it is.
const reading = computed(() => {
  if (lineStyle.value !== "readable") return undefined;
  const r = readFields(validValues.value);
  const lead = r.title ?? r.error ?? r.method ?? r.status;
  return lead === undefined ? undefined : r;
});

const shownValues = computed(() => {
  const used = reading.value?.used;
  if (!used || used.size === 0) return validValues.value;
  return Object.fromEntries(Object.entries(validValues.value).filter(([key]) => !used.has(key)));
});

// The line's own time and level repeat the timestamp column and the level marker,
// so they recede rather than go: the line still reads as it was written.
const isRepeated = (name: string | number) =>
  typeof name === "string" && (LEVEL_KEYS.includes(name) || (showTimestamp.value && TIME_KEYS.includes(name)));

const statusClass = (status: number) =>
  status >= 500 ? "text-error" : status >= 400 ? "text-warning" : status >= 300 ? "text-info" : "text-success";

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

.repeated {
  @apply opacity-40;

  .value {
    @apply font-normal;
  }
}

.headline {
  @apply flex flex-wrap gap-x-2 [overflow-wrap:anywhere];
}

.array {
  @apply before:text-base-content/80 after:text-base-content/80 before:content-['['] after:content-[']'];
}

.string {
  @apply before:content-['"'] after:content-['"'];
}
</style>
