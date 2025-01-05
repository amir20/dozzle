<template>
  <LogItem :logEntry @click="showDrawer(LogDetails, { entry: logEntry })" class="clickable">
    <ul class="space-x-4">
      <li v-for="(value, name) in validValues" :key="name" class="inline-flex">
        <span class="text-light">{{ name }}=</span><span class="font-bold" v-if="value === null">&lt;null&gt;</span>
        <template v-else-if="Array.isArray(value)">
          <span class="font-bold" v-html="JSON.stringify(value)"> </span>
        </template>
        <span class="font-bold" v-html="stripAnsi(value.toString())" v-else></span>
      </li>
      <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
    </ul>
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

const validValues = computed(() => {
  return Object.fromEntries(Object.entries(logEntry.message).filter(([_, value]) => value !== undefined));
});

const showDrawer = useDrawer();
</script>

<style scoped>
@import "@/main.css" reference;
.text-light {
  @apply text-base-content/70;
}
</style>
