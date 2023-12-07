<template>
  <div class="flex gap-x-2" @mouseover="isHovering = true" @mouseleave="isHovering = false">
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
      <ul class="fields cursor-pointer space-x-4" :class="{ expanded }">
        <li v-for="(value, name) in validValues" class="inline-block">
          <span class="text-light">{{ name }}=</span><span class="font-bold" v-if="value === null">&lt;null&gt;</span>
          <template v-else-if="Array.isArray(value)">
            <span class="font-bold" v-html="markSearch(JSON.stringify(value))"> </span>
          </template>
          <span class="font-bold" v-html="markSearch(value)" v-else></span>
        </li>
        <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
        <li class="inline-flex w-fit align-middle" v-show="isHovering">
          <div class="rounded px-1 py-1 hover:bg-slate-700 hover:text-secondary" @click="expandToggle()">
            <carbon:row-expand v-if="!expanded" />
            <carbon:row-collapse v-else />
          </div>
          <div
            @click="logEntry.copyLogMessageToClipBoard()"
            class="rounded px-1 py-1 hover:bg-slate-700 hover:text-secondary"
          >
            <carbon:copy-file />
          </div>
        </li>
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

const isHovering = ref(false);
</script>

<style lang="postcss" scoped>
.text-light {
  @apply text-base-content/70;
}

.context-actions {
  content: "expand json";
  @apply ml-2 inline-block;
  font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
}
</style>
