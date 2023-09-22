<template>
  <div class="flex gap-x-2">
    <div v-if="showStd">
      <log-std :std="logEntry.std"></log-std>
    </div>
    <div v-if="showTimestamp">
      <log-date :date="logEntry.date"></log-date>
    </div>
    <div class="flex">
      <log-level :level="logEntry.level"></log-level>
    </div>
    <div>
      <ul class="fields cursor-pointer space-x-4" :class="{ expanded }" @click="expandToggle()">
        <li v-for="(value, name) in validValues" class="inline-block">
          <span class="text-light">{{ name }}=</span><span class="font-bold" v-if="value === null">&lt;null&gt;</span>
          <template v-else-if="Array.isArray(value)">
            <span class="font-bold" v-html="markSearch(JSON.stringify(value))"> </span>
          </template>
          <span class="font-bold" v-html="markSearch(value)" v-else></span>
        </li>
        <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
      </ul>
      <field-list :fields="logEntry.unfilteredMessage" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { type ComplexLogEntry } from "@/models/LogEntry";

const { markSearch } = useSearchFilter();

const { logEntry } = defineProps<{
  logEntry: ComplexLogEntry;
  visibleKeys: string[][];
}>();

const [expanded, expandToggle] = useToggle();

const validValues = computed(() => {
  return Object.fromEntries(Object.entries(logEntry.message).filter(([_, value]) => value !== undefined));
});
</script>

<style lang="postcss" scoped>
.text-light {
  @apply text-base-content/70;
}
.fields {
  &:hover {
    &::after {
      content: "expand json";
      @apply ml-2 inline-block text-secondary;
      font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
    }
  }

  &.expanded:hover {
    &::after {
      content: "collapse json";
    }
  }
}
</style>
