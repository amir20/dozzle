<template>
  <DefineTemplate v-slot="{ data }">
    <ul class="inline-flex space-x-4" @click="preventDefaultOnLinks">
      <li v-for="(value, name) in data" :key="name" v-if="isObject(data)">
        <span class="text-light">{{ name }}=</span>
        <span class="font-bold" v-if="value === null">&lt;null&gt;</span>
        <span v-else-if="Array.isArray(value)" class="font-bold">
          [<span v-for="(item, index) in value" :key="index">
            <ReuseTemplate :data="item" v-if="isObject(item) || Array.isArray(item)" />
            <span v-else class="font-bold" v-html="stripAnsi(item.toString())"></span>
            <span v-if="index < value.length - 1">, </span></span
          >]
        </span>
        <span v-else class="red font-bold" v-html="stripAnsi(value.toString())"></span>
      </li>
      <li v-else-if="Array.isArray(data)">
        [<span v-for="(item, index) in data" :key="index">
          <ReuseTemplate :data="item" v-if="isObject(item) || Array.isArray(item)" />
          <span v-else class="font-bold" v-html="stripAnsi(item.toString())"></span>
          <span v-if="index < data.length - 1">, </span></span
        >]
      </li>
      <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
    </ul>
  </DefineTemplate>
  <LogItem :logEntry>
    <div @click="containers.length > 0 && showDrawer(LogDetails, { entry: logEntry })" class="cursor-pointer">
      <ReuseTemplate :data="validValues"></ReuseTemplate>
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
.text-light {
  @apply text-base-content/70;
}
</style>
