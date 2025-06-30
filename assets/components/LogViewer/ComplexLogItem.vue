<template>
  <DefineTemplate v-slot="{ data }">
    <ul class="inline-flex flex-wrap space-x-4" @click="preventDefaultOnLinks">
      <li v-for="(value, name) in data" :key="name" v-if="isObject(data)">
        <span class="key">{{ name }}=</span>
        <span class="value" v-if="value === null">&lt;null&gt;</span>
        <ReuseTemplate :data="value" v-else-if="isObject(value) || Array.isArray(value)" />
        <span v-else class="value" :class="typeof value" v-html="stripAnsi(value.toString())"></span>
      </li>
      <li v-else-if="Array.isArray(data)">
        <ul class="array inline-flex flex-wrap space-x-1">
          <li
            v-for="(item, index) in data"
            :key="index"
            class="after:text-base-content/70 not-last:after:content-[',']"
          >
            <ReuseTemplate :data="item" v-if="isObject(item) || Array.isArray(item)" />
            <span v-else class="value" :class="typeof item" v-html="stripAnsi(item.toString())"></span>
          </li>
        </ul>
      </li>
      <li class="key" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
    </ul>
  </DefineTemplate>
  <LogItem :logEntry>
    <div @click="containers.length > 0 && showDrawer(LogDetails, { entry: logEntry })" class="cursor-pointer">
      <ReuseTemplate :data="validValues" />
    </div>
  </LogItem>
</template>
<script lang="ts" setup>
import stripAnsi from "strip-ansi";
import { type ComplexLogEntry } from "@/models/LogEntry";
import LogDetails from "./LogDetails.vue";

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
  @apply text-base-content font-bold;
}

.array {
  @apply before:text-base-content/80 after:text-base-content/80 before:content-['['] after:content-[']'];
}

.string {
  @apply before:content-['"'] after:content-['"'];
}
</style>
